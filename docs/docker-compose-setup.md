# Docker Compose Reference Architecture

This document describes the complete `docker-compose.yaml` setup for running the DevSecOps demo stack locally using Docker Compose.

---

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     Docker Compose Network                    │
│                                                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐ │
│  │ postgres │  │ backend  │  │ frontend │  │     lgtm     │ │
│  │  :5432   │  │  :8000   │  │  :8080   │  │  :3000 (UI)  │ │
│  └──────────┘  └──────────┘  └──────────┘  │  :4317 (gRPC)│ │
│                      │              │       │  :4318 (HTTP)│ │
│                      │              │       └──────────────┘ │
│                      │              │              ▲         │
│                      │              │              │         │
│                      ▼              ▼              │         │
│                 ┌──────────────────────────────────┘         │
│                 │           alloy (collector)                │
│                 │  :4317 (gRPC) :4318 (HTTP) :4040 (Prof)   │
│                 └────────────────────────────────────────────┘
└──────────────────────────────────────────────────────────────┘
```

---

## docker-compose.yaml

```yaml
version: "3.9"

services:
  # -----------------------------------------------------------------------
  # PostgreSQL
  # -----------------------------------------------------------------------
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: items
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "postgres"]
      interval: 5s
      timeout: 3s
      retries: 5

  # -----------------------------------------------------------------------
  # Grafana LGTM (All-in-One Observability)
  # -----------------------------------------------------------------------
  lgtm:
    image: grafana/otel-lgtm:latest
    ports:
      - "3000:3000"   # Grafana UI
      - "3100:3100"   # Loki
      - "9090:9090"   # Mimir (Prometheus)
    volumes:
      - lgtm-data:/data

  # -----------------------------------------------------------------------
  # Grafana Alloy (OTel Collector)
  # -----------------------------------------------------------------------
  alloy:
    image: grafana/alloy:latest
    command:
      - run
      - /etc/alloy/config.alloy
      - --server.http.listen-addr=0.0.0.0:12345
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
      - "4040:4040"   # Pyroscope
      - "12345:12345" # Alloy UI
    volumes:
      - ../config.alloy:/etc/alloy/config.alloy:ro
    depends_on:
      - lgtm

  # -----------------------------------------------------------------------
  # Backend (Python FastAPI)
  # -----------------------------------------------------------------------
  backend:
    build:
      context: ../backend
      dockerfile: Dockerfile
    image: docker.io/moreskylab/backend:1.0.0
    ports:
      - "8000:8000"
    environment:
      DATABASE_URL: postgresql://postgres:postgres@postgres:5432/items
      ALLOWED_ORIGIN: http://localhost:8080
      ENABLE_OBSERVABILITY: "true"
      OTEL_EXPORTER_OTLP_ENDPOINT: http://alloy:4317
      PYROSCOPE_SERVER: http://alloy:4040
      WEB_CONCURRENCY: "2"
      PORT: "8000"
    depends_on:
      postgres:
        condition: service_healthy
      alloy:
        condition: service_started

  # -----------------------------------------------------------------------
  # Frontend (Nginx + Vite build)
  # -----------------------------------------------------------------------
  frontend:
    build:
      context: ../frontend
      dockerfile: Dockerfile
    image: docker.io/moreskylab/frontend:1.0.0
    ports:
      - "8080:8080"
    depends_on:
      - backend
      - alloy

volumes:
  postgres-data:
  lgtm-data:
```

---

## Usage

```bash
# Start the full stack
docker compose -f docs/docker-compose.yaml up -d --build

# View logs
docker compose -f docs/docker-compose.yaml logs -f

# Stop
docker compose -f docs/docker-compose.yaml down

# Stop and remove volumes
docker compose -f docs/docker-compose.yaml down -v
```

---

## Access Points

| Service   | URL                          | Credentials        |
|-----------|------------------------------|---------------------|
| Frontend  | http://localhost:8080         | —                   |
| Backend   | http://localhost:8000/docs    | —                   |
| Grafana   | http://localhost:3000         | admin / admin       |
| Alloy UI  | http://localhost:12345        | —                   |
