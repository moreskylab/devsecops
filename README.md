# DevSecOps Demo

A **full-stack, cloud-native DevSecOps demo application** demonstrating an end-to-end observability pipeline with a complete CI/CD security toolchain.

CRUD "Item Manager" app with deep **OpenTelemetry** instrumentation across every layer — from the browser to the database.

---

## Architecture

```
Frontend (Vite+TS) → NGINX (Reverse Proxy) → Backend (FastAPI/Gin) → PostgreSQL
       ↓                    ↓                        ↓
    OTel HTTP           /items → backend          OTel gRPC
    /v1/* → alloy                                + Pyroscope
       ↓                                            ↓
              Grafana Alloy (OTel Collector)
              ↓           ↓          ↓          ↓
           Tempo        Loki       Mimir     Pyroscope
                    Grafana Dashboard (:3000)
```

---

## Quick Start

### Prerequisites

- [Podman](https://podman.io/) or [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) (for K8s deployment)

### Local Development (Podman)

```bash
# Build images
cd backend && podman build -t docker.io/moreskylab/backend:1.0.0 .
cd ../frontend && podman build -t docker.io/moreskylab/frontend:1.0.0 .

# Run the stack
cd ../local-podman-setup
podman play kube deployment.yaml
```

### Access

| Service   | URL                    |
|-----------|------------------------|
| Frontend  | http://localhost:8080   |
| Backend   | http://localhost:8000   |
| Grafana   | http://localhost:3000   |
| Alloy UI  | http://localhost:12345  |

---

## Tech Stack

| Layer            | Technology                                         |
|------------------|----------------------------------------------------|
| **Frontend**     | Vanilla TypeScript, Vite, OTel Browser SDK         |
| **Backend**      | FastAPI (Python) / Gin (Go), SQLModel / GORM       |
| **Database**     | PostgreSQL 17 (Alpine)                             |
| **Serving**      | Nginx Unprivileged 1.27                            |
| **Observability**| Grafana Alloy → Tempo, Loki, Mimir, Pyroscope     |
| **CI/CD**        | Jenkins + Hadolint, Trivy, Gitleaks, Semgrep, Cosign, Grype, OWASP ZAP |
| **Containers**   | Multi-stage builds → Distroless runtime images     |
| **Orchestration**| Kubernetes (K3s) / Podman                          |

---

## Project Structure

```
├── backend/                # Python FastAPI backend
├── frontend/               # Vanilla TypeScript frontend
├── production/backend/     # Go (Gin) production variant
├── k8s/                    # Kubernetes manifests
├── local-podman-setup/     # Podman local dev setup
├── docs/                   # Jenkinsfile, runbook, compose reference
└── config.alloy            # Grafana Alloy pipeline config
```

---

## Documentation

- [Operational Runbook](docs/README.md) — kubectl, podman, helm, argocd commands
- [Docker Compose Setup](docs/docker-compose-setup.md) — Full compose reference
- [CI/CD Pipeline](docs/Jenkinsfile) — Jenkins declarative pipeline
- [Local Podman Setup](local-podman-setup/README.md) — Quick start with Podman

---

## API Endpoints

| Method   | Path               | Description              |
|----------|--------------------|--------------------------|
| `GET`    | `/healthz`         | Liveness probe           |
| `GET`    | `/readyz`          | Readiness probe          |
| `POST`   | `/items`           | Create item              |
| `GET`    | `/items`           | List items (paginated)   |
| `PUT`    | `/items/{id}`      | Update item              |
| `DELETE` | `/items/{id}`      | Delete item              |

---

## Container Images

```
docker.io/moreskylab/backend:1.0.0
docker.io/moreskylab/frontend:1.0.0
```

---

## License

See [LICENSE](LICENSE) for details.