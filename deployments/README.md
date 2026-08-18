# Deployments

This directory contains deployment configurations for various platforms.

## Docker

### Quick Start

```bash
# Build and run
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

### Configuration

Edit `docker-compose.yml` or use environment variables in `.env` file:

```env
ANOIVE_API_KEY=your-secure-api-key
PULSE_LOG_LEVEL=info
PULSE_LOG_FORMAT=json
PULSE_RETENTION_HOURS=168
```

### Volumes

- `anohive-data`: Persistent database storage
- `app-logs`: Application logs (when using collector profile)

### Profiles

- `default`: API server only
- `collector`: API server + log collector sidecar

```bash
docker-compose --profile collector up -d
```

## Kubernetes

### Quick Start

```bash
# Deploy all resources
kubectl apply -k deployments/kubernetes/

# Check status
kubectl -n anohive get pods

# Port forward for local access
kubectl -n anohive port-forward svc/anohive 8080:80
```

### Directory Structure

```
deployments/kubernetes/
├── kustomization.yaml      # Kustomization base
├── namespace.yaml          # Namespace definition
├── configmap.yaml          # Configuration and secrets
├── pvc.yaml                # Persistent volume claim
├── deployment.yaml         # Main deployment
├── service.yaml            # Service and ingress
├── hpa.yaml                # Horizontal pod autoscaler
└── networkpolicy.yaml      # Network security policy
```

### Configuration

1. Edit `configmap.yaml` to set your configuration
2. Update `deployment.yaml` to set your image tag
3. Modify `service.yaml` to configure ingress

### Scaling

```bash
# Scale manually
kubectl -n anohive scale deployment/anohive --replicas=3

# Or rely on HPA (configured in hpa.yaml)
kubectl -n anohive get hpa
```

### Monitoring

```bash
# Check pod logs
kubectl -n anohive logs -f deployment/anohive

# Check resource usage
kubectl -n anohive top pods

# Check events
kubectl -n anohive get events
```

### Upgrades

```bash
# Update image
kubectl -n anohive set image deployment/anohive anohive=anohive/anohive:v1.2.0

# Check rollout status
kubectl -n anohive rollout status deployment/anohive

# Rollback if needed
kubectl -n anohive rollout undo deployment/anohive
```

### Backup

```bash
# Create backup pod
kubectl -n anohive run anohive-backup --rm -i --restart=Never \
  --image=anohive/anohive:latest \
  --overrides='{"spec": {"containers": [{"name": "backup", "image": "anohive/anohive:latest", "command": ["anohive-cli", "backup", "-db", "/var/lib/anohive/data/anohive.db", "-output", "/backup/anohive-backup.db"], "volumeMounts": [{"name": "data", "mountPath": "/var/lib/anohive/data"}, {"name": "backup", "mountPath": "/backup"}]}], "volumes": [{"name": "data", "persistentVolumeClaim": {"claimName": "anohive-data"}}, {"name": "backup", "emptyDir": {}}]}}'
```

## CI/CD

GitHub Actions workflows are configured in `.github/workflows/`:

- `ci.yml`: Continuous Integration (lint, test, build)
- `cd.yml`: Continuous Deployment (release, deploy)

### Required Secrets

- `GITHUB_TOKEN`: Automatically provided by GitHub Actions

### Deployment Environments

- `staging`: Auto-deployed on push to main
- `production`: Deployed on tag push (v*.*.*)
