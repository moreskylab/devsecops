# Local Podman Setup

Run the entire DevSecOps demo stack locally using Podman.

## Prerequisites

- [Podman](https://podman.io/) installed
- Container images built or available:
  - `docker.io/moreskylab/backend:1.0.0`
  - `docker.io/moreskylab/frontend:1.0.0`

## Quick Start

```bash
# From this directory (local-podman-setup/)
podman play kube deployment.yaml
```

## Access Points

| Service   | URL                          |
|-----------|------------------------------|
| Frontend  | http://localhost:8080         |
| Backend   | http://localhost:8000         |
| Grafana   | http://localhost:3000         |
| Alloy UI  | http://localhost:12345        |

## Building Images Locally

```bash
# Backend
cd ../backend
podman build -t docker.io/moreskylab/backend:1.0.0 .

# Frontend
cd ../frontend
podman build -t docker.io/moreskylab/frontend:1.0.0 .
```

## Stopping

```bash
podman play kube --down deployment.yaml
```

## Troubleshooting

```bash
# View pod status
podman pod ps

# View container logs
podman logs devsecops-backend
podman logs devsecops-frontend
podman logs devsecops-postgres
podman logs devsecops-lgtm
podman logs devsecops-alloy
```
