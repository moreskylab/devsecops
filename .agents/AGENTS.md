# DevSecOps Demo — AI Assistant System Prompt

> **Purpose**: This file provides full project context for LLM-powered coding assistants (GitHub Copilot, Cursor, etc.) so they can generate accurate, architecture-aware code and suggestions.

---

## 1. Project Overview

This is a **full-stack, cloud-native DevSecOps demo application** that demonstrates an end-to-end observability pipeline with a complete CI/CD security toolchain. It is a CRUD "Item Manager" app with deep OpenTelemetry instrumentation across every layer.

**Core value proposition**: Demonstrate how to instrument, secure, and deploy a modern microservices application with full distributed tracing, metrics, logs, and continuous profiling — from the browser all the way down to the database.

---

## 2. Architecture

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│  Frontend    │────▶│   NGINX      │────▶│  Backend   │
│  (Vite+TS)  │     │  (Reverse    │     │  (FastAPI / │
│  Port 8080  │     │   Proxy)     │     │   Gin+Go)  │
└──────┬──────┘     └──────┬───────┘     └─────┬──────┘
       │                   │                   │
       │ OTel Traces/      │ /items → backend  │ OTel via gRPC
       │ Metrics/Logs      │ /v1/* → alloy     │ + Pyroscope
       ▼                   ▼                   ▼
┌──────────────┐     ┌──────────────┐     ┌────────────┐
│ Grafana Alloy│────▶│  Grafana     │     │ PostgreSQL │
│ (OTel       │     │  LGTM Stack  │     │  (alpine)  │
│  Collector) │     │  (Loki,Tempo,│     │  Port 5432 │
│ Ports:      │     │   Mimir,     │     └────────────┘
│  4317(gRPC) │     │   Pyroscope, │
│  4318(HTTP) │     │   Grafana)   │
│  4040(Prof) │     │  Port 3000   │
└─────────────┘     └──────────────┘
```

### Deployment Targets
| Environment         | Orchestration       | Ingress / Tunnel        |
|---------------------|---------------------|-------------------------|
| Local Development   | Podman / Docker     | localhost port mapping   |
| Local K8s (Podman)  | `podman play kube`  | Port forwarding          |
| Production K8s      | Kubernetes (K3s)    | ngrok Gateway API or Pangolin/Newt |
| CI/CD               | Jenkins Declarative | Docker build + push      |

---

## 3. Technology Stack

### Backend (Primary — Python)
| Component          | Technology                                         |
|--------------------|----------------------------------------------------|
| Framework          | **FastAPI** with Uvicorn (ASGI)                    |
| ORM / Models       | **SQLModel** (Pydantic + SQLAlchemy hybrid)        |
| Database           | **PostgreSQL** (Alpine image, `psycopg2-binary`)   |
| Package Manager    | **uv** (with `pyproject.toml` + `uv.lock`)         |
| Python Version     | **3.12+**                                          |
| Tracing            | OpenTelemetry SDK + OTLP gRPC exporters            |
| Metrics            | OpenTelemetry SDK (`PeriodicExportingMetricReader`) |
| Logging            | OpenTelemetry `LoggerProvider` + `LoggingInstrumentor` |
| Profiling          | Pyroscope (`pyroscope-io`, Linux-only)             |
| Container          | Multi-stage: `uv:python3.12-bookworm-slim` → `gcr.io/distroless/python3-debian12` |
| Auto-Instrumentation | `FastAPIInstrumentor`, `SQLAlchemyInstrumentor`, `LoggingInstrumentor` |

### Backend (Production variant — Go)
| Component          | Technology                                         |
|--------------------|----------------------------------------------------|
| Framework          | **Gin** (gin-gonic) with `gin-contrib/cors`        |
| ORM                | **GORM** with `gorm.io/driver/postgres`            |
| Tracing/Metrics    | OpenTelemetry Go SDK + OTLP gRPC exporters         |
| DB Instrumentation | `gorm.io/plugin/opentelemetry/tracing`             |
| Profiling          | `grafana/pyroscope-go`                             |
| Container          | Multi-stage: `golang:1.26.4-trixie` → `gcr.io/distroless/base-debian13` (pinned digest) |

### Frontend
| Component          | Technology                                         |
|--------------------|----------------------------------------------------|
| Framework          | **Vanilla TypeScript** (no React/Vue/Angular)      |
| Build Tool         | **Vite 8** with TypeScript 6                       |
| Tracing            | `@opentelemetry/sdk-trace-web` + OTLP HTTP exporter |
| Metrics            | `@opentelemetry/sdk-metrics` + OTLP HTTP exporter  |
| Logging            | `@opentelemetry/sdk-logs` + OTLP HTTP exporter     |
| Instrumentation    | `FetchInstrumentation`, `UserInteractionInstrumentation` |
| Serving            | **nginxinc/nginx-unprivileged:1.27-alpine** (UID 101) |
| Dev Server Port    | 8080                                               |

### Observability Stack
| Component        | Image                          | Purpose                        |
|------------------|--------------------------------|--------------------------------|
| Grafana Alloy    | `grafana/alloy:latest`         | Unified OTel Collector/Agent   |
| Grafana LGTM     | `grafana/otel-lgtm:latest`     | All-in-one: Grafana + Loki + Tempo + Mimir + Pyroscope |
| Grafana (standalone) | `grafana/grafana:11.0.0`   | Dashboard (docker-compose mode)|
| Loki             | `grafana/loki:3.0.0`           | Log aggregation (standalone)   |
| Tempo            | `grafana/tempo:2.4.1`          | Distributed tracing (standalone)|
| Mimir            | `grafana/mimir:2.12.0`         | Metrics (Prometheus-compatible)|
| Pyroscope        | `grafana/pyroscope:1.6.0`      | Continuous profiling           |

### CI/CD & DevSecOps Toolchain (Jenkins)
| Tool       | Purpose                                                |
|------------|--------------------------------------------------------|
| Hadolint   | Dockerfile linting                                     |
| Trivy      | Container image vulnerability scanning + SBOM generation |
| Gitleaks   | Git secrets detection                                  |
| Semgrep    | Static Application Security Testing (SAST)             |
| Cosign     | Container image signing & attestation (via Vault transit key) |
| Grype      | Software Composition Analysis (SCA)                    |
| OWASP ZAP  | Dynamic Application Security Testing (DAST)            |
| Vault      | Secrets management (HashiCorp Vault)                   |
| Slack      | Build notification alerts                              |

---

## 4. Directory Structure

```
devsecops-demo/
├── backend/                    # Python FastAPI backend (primary)
│   ├── main.py                 # Single-file FastAPI app with full OTel instrumentation
│   ├── Dockerfile              # Multi-stage: uv builder → distroless runtime
│   ├── pyproject.toml          # Python deps managed by uv
│   ├── requirements.txt        # Fallback pip requirements
│   ├── uv.lock                 # Lockfile for deterministic builds
│   └── .env                    # Runtime env vars (DB, OTel, Pyroscope endpoints)
│
├── frontend/                   # Vanilla TypeScript frontend
│   ├── index.html              # Entry HTML with CRUD form
│   ├── src/
│   │   ├── main.ts             # App logic + full OTel (traces, metrics, logs)
│   │   └── style.css           # CSS with custom properties / design tokens
│   ├── Dockerfile              # Multi-stage: node builder → nginx-unprivileged
│   ├── nginx-secure.conf       # Production Nginx: security headers + CSP + reverse proxy
│   ├── nginx-insecure.conf     # Dev/demo Nginx: permissive CORS + dynamic DNS resolver
│   ├── vite.config.ts          # Vite config (port 8080)
│   ├── tsconfig.json           # TypeScript strict config (ES2023, bundler resolution)
│   ├── package.json            # npm deps (all OTel browser packages)
│   └── .env                    # Vite build-time env vars
│
├── production/                 # Production-ready Go backend rewrite
│   ├── backend/
│   │   ├── main.go             # Gin-based Go backend (identical API surface)
│   │   ├── Dockerfile          # Multi-stage: Go builder → distroless (pinned digest)
│   │   ├── go.mod              # Go module definition
│   │   └── go.sum              # Dependency checksums
│   └── frontend/               # (empty — uses same frontend)
│
├── k8s/                        # Kubernetes manifests
│   ├── all-in-one.yaml         # Full stack: namespace, PVCs, secrets, configmaps,
│   │                           #   postgres StatefulSet, LGTM, Alloy, backend, frontend,
│   │                           #   Gateway API (ngrok), HTTPRoutes
│   ├── all-in-one-cilium.yaml  # Variant with Cilium networking
│   ├── all-in-one-cilium-prod.yaml  # Production Cilium variant
│   ├── all-in-one-pangolin.yaml     # Variant using Pangolin/Newt tunnel
│   └── local.yaml              # Simplified local K8s manifest
│
├── local-podman-setup/         # Podman-based local development
│   ├── deployment.yaml         # Kubernetes Pod YAML for `podman play kube`
│   ├── config.alloy            # Alloy config embedded as ConfigMap
│   ├── nginx.conf              # Local nginx config
│   └── README.md               # Podman launch command
│
├── docs/                       # Documentation & reference configs
│   ├── README.md               # Operational runbook (podman, kubectl, helm, argocd)
│   ├── Jenkinsfile             # Full DevSecOps CI/CD pipeline
│   ├── docker-compose-setup.md # Complete docker-compose reference architecture
│   ├── Dockerfile.backend      # Alternate backend Dockerfile
│   ├── cloudflare-tunnel.yaml  # Cloudflare tunnel k8s manifest
│   └── ngrok-tunnel.yaml       # ngrok tunnel k8s manifest
│
├── config.alloy                # Root-level Grafana Alloy pipeline config
├── .gitignore                  # Combined Node.js + Python gitignore
└── README.md                   # Project README
```

---

## 5. Key API Surface

All endpoints served on port **8000** (backend):

| Method   | Path              | Description                  | Status Codes       |
|----------|-------------------|------------------------------|--------------------|
| `GET`    | `/healthz`        | Kubernetes liveness probe    | 200                |
| `GET`    | `/readyz`         | Kubernetes readiness probe   | 200, 503           |
| `POST`   | `/items`          | Create a new item            | 201                |
| `GET`    | `/items`          | List items (paginated: `skip`, `limit`) | 200    |
| `PUT`    | `/items/{item_id}`| Update an item               | 200, 404           |
| `DELETE` | `/items/{item_id}`| Delete an item               | 204, 404           |

### Data Models
```python
# SQLModel schema (Python)
class Item:
    id: int          # Auto-increment PK
    title: str       # Indexed
    description: str | None
```

```go
// GORM schema (Go)
type Item struct {
    ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
    Title       string `gorm:"index" json:"title"`
    Description string `json:"description"`
}
```

---

## 6. Conventions & Patterns

### Code Style
- **Python**: Type hints everywhere. Use `Annotated[Session, Depends(get_db)]` pattern for DI. SQLModel for data models. Async lifespan manager.
- **Go**: Gin handlers follow `func(c *gin.Context)` pattern. GORM with context propagation (`db.WithContext()`). Structured logging via `slog`.
- **TypeScript**: Strict mode. No frameworks — vanilla DOM manipulation. `DocumentFragment` for batch rendering. Event delegation over individual listeners.

### Observability Conventions
- **Service name**: `"backend"` for backend, `"frontend"` for frontend
- **Service version**: `"1.0.0"`
- **Sampling**: 10% trace sampling (`TraceIdRatioBased(0.1)`)
- **Custom spans**: Named `ui.submit_item`, `ui.delete_item`, `ui.render_list`, `fetch_all_items_from_db`
- **Custom metrics**: `items_created_total` counter, `ui.operation.duration` histogram
- **Metric labels**: Use `status: "success"/"error"` — avoid high-cardinality labels (no item titles)
- **Health endpoints excluded** from instrumentation: `/healthz`, `/readyz`, `/metrics`

### Container Conventions
- **Multi-stage builds** — always. Builder stage → minimal runtime stage.
- **Distroless runtime images** for backend (Python: `gcr.io/distroless/python3-debian12`, Go: `gcr.io/distroless/base-debian13`)
- **Non-root execution**: `USER nonroot` (distroless) or `USER 101` (nginx-unprivileged)
- **No shell in production containers** — use JSON array `ENTRYPOINT`/`CMD` syntax

### Kubernetes Conventions
- **Namespace**: `observability`
- **Probes**: `httpGet` liveness on `/healthz`, readiness on `/healthz` or `/readyz`
- **Resource limits**: Always set `requests` and `limits` for CPU/memory
- **Secrets**: Kubernetes `Secret` objects for DB credentials; `ConfigMap` for non-sensitive config
- **Rolling updates**: `maxSurge: 1, maxUnavailable: 0`
- **InitContainers**: `wait-for-postgres` pattern using `pg_isready`
- **Gateway API**: Using `gateway.networking.k8s.io/v1` (`GatewayClass`, `Gateway`, `HTTPRoute`)

### Security Conventions
- **CORS**: Strict `ALLOWED_ORIGIN` env var (not `*` in production)
- **CSP headers**: `Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`
- **Server tokens**: `server_tokens off` in Nginx
- **W3C Trace Context propagation** across frontend → nginx → backend
- **Cosign** image signing with Vault transit keys
- **Trivy** scanning with CycloneDX SBOM generation

### Environment Variables

**Backend** (Python):
| Variable                       | Purpose                                | Default                          |
|--------------------------------|----------------------------------------|----------------------------------|
| `DATABASE_URL`                 | PostgreSQL connection string           | `sqlite:///./app.db`             |
| `ALLOWED_ORIGIN`               | CORS allowed origin                    | `http://localhost:8000`          |
| `ENABLE_OBSERVABILITY`         | Toggle OTLP vs console exporters       | `false`                          |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint (gRPC)         | (OTel SDK default)              |
| `PYROSCOPE_SERVER`             | Pyroscope/Alloy profiling endpoint     | `http://alloy:4040`             |
| `WEB_CONCURRENCY`             | Uvicorn worker count                   | `4`                              |
| `PORT`                         | Application listen port                | `8000`                           |

**Frontend** (Vite build-time):
| Variable               | Purpose                                | Default  |
|------------------------|----------------------------------------|----------|
| `VITE_ENABLE_TRACKING` | Enable/disable OTel in browser         | `true`   |
| `VITE_APP_VERSION`     | Application version for OTel resource  | `1.0.0`  |

---

## 7. Telemetry Pipeline Flow

```
Frontend (Browser)                    Backend (Server)
      │                                     │
      │ OTLP/HTTP (/v1/traces,             │ OTLP/gRPC (:4317)
      │  /v1/logs, /v1/metrics)            │ + Pyroscope HTTP (:4040)
      │                                     │
      ▼                                     ▼
┌──────────────────────────────────────────────────┐
│              Grafana Alloy (Collector)            │
│  ┌────────────────────────────────────────────┐  │
│  │ otelcol.receiver.otlp "default"            │  │
│  │  gRPC :4317  |  HTTP :4318                 │  │
│  └───────────────┬────────────────────────────┘  │
│  ┌───────────────┴────────────────────────────┐  │
│  │ Exporters:                                 │  │
│  │  Metrics → prometheus.remote_write → Mimir │  │
│  │  Traces  → otelcol.exporter.otlp  → Tempo │  │
│  │  Logs    → otelcol.exporter.otlp  → Loki  │  │
│  │  Profiles→ pyroscope.write         → Pyro  │  │
│  └────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────┐
│              Grafana LGTM Stack                   │
│   Grafana :3000  |  Tempo :4317  |  Loki :3100   │
│   Mimir :9090    |  Pyroscope :4040              │
└──────────────────────────────────────────────────┘
```

---

## 8. CI/CD Pipeline Stages (Jenkins)

```
Initialize & Notify (Slack)
        │
Vault Integration Test
        │
Global Security Gate: Secrets (Gitleaks)
        │
┌───────┴───────┐
│   Parallel    │
├───────────────┤
│ Frontend      │  Backend
│ Track:        │  Track:
│  ├─SAST       │   ├─SAST (Semgrep)
│  ├─Lint       │   ├─Lint (Hadolint)
│  ├─SCA(Grype) │   ├─SCA (Grype)
│  ├─Build      │   ├─Build (Docker)
│  └─Trivy Scan │   └─Trivy Scan + SBOM
└───────┬───────┘
        │
DAST (OWASP ZAP) — main branch only
        │
Publish & Sign (Docker Push + Cosign + Attestations)
        │
Deploy to Kubernetes (kubectl apply + rollout status)
        │
Post: Cleanup + Slack Notifications (success/failure)
```

---

## 9. Guidelines for AI Assistants

1. **When generating backend code**: Use FastAPI + SQLModel patterns. Always include type hints. Follow the existing `SessionDep = Annotated[Session, Depends(get_db)]` DI pattern. Wrap database queries in OTel spans.

2. **When generating frontend code**: Write vanilla TypeScript — NO React, Vue, or Angular. Use DOM APIs directly. Instrument all fetch calls with OTel spans. Use `DocumentFragment` for efficient list rendering.

3. **When modifying Dockerfiles**: Maintain multi-stage builds. Always use distroless or unprivileged base images. Never add a shell. Use `USER nonroot`.

4. **When writing K8s manifests**: Use namespace `observability`. Include resource requests/limits. Add health probes. Use Secrets for credentials, ConfigMaps for config.

5. **When adding new endpoints**: Include them in NGINX proxy config, Kubernetes HTTPRoute, and Alloy CORS config if browser-facing.

6. **When touching telemetry**: Follow the existing metric naming conventions. Avoid high-cardinality metric labels. Maintain the 10% trace sampling rate.

7. **When writing tests or CI steps**: Follow the Jenkinsfile pattern — parallel stages, security scanning before build, signing after push.

8. **Container registry**: `docker.io/moreskylab/{backend,frontend}:TAG`

9. **Image tagging**: `{BUILD_NUMBER}-{GIT_SHORT_SHA}` in CI; `1.0.0` for manual builds.

10. **Go backend**: The `production/backend/` Go variant is API-compatible with the Python backend. Both use identical endpoint paths, request/response schemas, and OTel instrumentation patterns. Prefer Go for production performance; Python for rapid iteration.
