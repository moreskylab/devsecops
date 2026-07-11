# DevSecOps Demo — Operational Runbook

This document covers day-to-day operations for the DevSecOps demo application.

---

## Table of Contents

- [Local Development (Podman)](#local-development-podman)
- [Kubernetes (kubectl)](#kubernetes-kubectl)
- [Helm](#helm)
- [ArgoCD](#argocd)
- [Container Image Management](#container-image-management)
- [Observability Access](#observability-access)

---

## Local Development (Podman)

### Start the full stack

```bash
cd local-podman-setup/
podman play kube deployment.yaml
```

### Stop the full stack

```bash
podman play kube --down deployment.yaml
```

### Build images locally

```bash
# Backend (Python)
cd backend/
podman build -t docker.io/moreskylab/backend:1.0.0 .

# Backend (Go — production variant)
cd production/backend/
podman build -t docker.io/moreskylab/backend:1.0.0-go .

# Frontend
cd frontend/
podman build -t docker.io/moreskylab/frontend:1.0.0 .
```

### View logs

```bash
podman logs -f devsecops-backend
podman logs -f devsecops-frontend
podman logs -f devsecops-postgres
```

---

## Kubernetes (kubectl)

### Deploy the full stack

```bash
# All-in-one (production)
kubectl apply -f k8s/all-in-one.yaml

# Local development
kubectl apply -f k8s/local.yaml
```

### Check deployment status

```bash
kubectl -n observability get pods
kubectl -n observability get svc
kubectl -n observability get deploy
```

### Wait for rollout

```bash
kubectl -n observability rollout status deployment/backend
kubectl -n observability rollout status deployment/frontend
```

### View logs

```bash
kubectl -n observability logs -f deployment/backend
kubectl -n observability logs -f deployment/frontend
kubectl -n observability logs -f deployment/alloy
```

### Port forwarding (local access)

```bash
# Frontend
kubectl -n observability port-forward svc/frontend 8080:8080

# Grafana
kubectl -n observability port-forward svc/lgtm 3000:3000

# Backend (direct)
kubectl -n observability port-forward svc/backend 8000:8000
```

### Restart a deployment

```bash
kubectl -n observability rollout restart deployment/backend
kubectl -n observability rollout restart deployment/frontend
```

### Scale

```bash
kubectl -n observability scale deployment/backend --replicas=3
kubectl -n observability scale deployment/frontend --replicas=3
```

### Delete the stack

```bash
kubectl delete -f k8s/all-in-one.yaml
```

---

## Helm

> Note: Helm charts are not yet included in this repo. Below are reference commands for when they're added.

```bash
# Install
helm install devsecops ./helm/devsecops -n observability --create-namespace

# Upgrade
helm upgrade devsecops ./helm/devsecops -n observability

# Uninstall
helm uninstall devsecops -n observability
```

---

## ArgoCD

> Reference commands for GitOps-based deployment.

```bash
# Create ArgoCD application
argocd app create devsecops \
  --repo https://github.com/moreskylab/devsecops.git \
  --path k8s \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace observability \
  --sync-policy automated

# Sync manually
argocd app sync devsecops

# Check status
argocd app get devsecops
```

---

## Container Image Management

### Push images

```bash
podman push docker.io/moreskylab/backend:1.0.0
podman push docker.io/moreskylab/frontend:1.0.0
```

### Image tagging convention

| Context        | Tag Format                      | Example          |
|----------------|---------------------------------|------------------|
| CI/CD (Jenkins)| `{BUILD_NUMBER}-{GIT_SHORT_SHA}`| `42-a1b2c3d`     |
| Manual builds  | `1.0.0`                         | `1.0.0`          |
| Development    | `dev-{date}`                    | `dev-20250101`   |

---

## Observability Access

| Service     | Local URL                     | K8s Port-Forward         |
|-------------|-------------------------------|--------------------------|
| Grafana     | http://localhost:3000          | `svc/lgtm 3000:3000`    |
| Alloy UI    | http://localhost:12345         | `svc/alloy 12345:12345`  |
| Frontend    | http://localhost:8080          | `svc/frontend 8080:8080` |
| Backend API | http://localhost:8000          | `svc/backend 8000:8000`  |

### Default Grafana credentials

- **Username**: `admin`
- **Password**: `admin`

### Key Grafana data sources (auto-provisioned by LGTM)

| Data Source  | Type       | URL                    |
|-------------|------------|------------------------|
| Tempo       | Tempo      | http://localhost:3200   |
| Loki        | Loki       | http://localhost:3100   |
| Mimir       | Prometheus | http://localhost:9090   |
| Pyroscope   | Pyroscope  | http://localhost:4040   |
