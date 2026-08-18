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
PULSE_API_KEY=your-secure-api-key
PULSE_LOG_LEVEL=info
PULSE_LOG_FORMAT=json
PULSE_RETENTION_HOURS=168
```

### Volumes

- `pulse-data`: Persistent database storage
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
kubectl -n pulse-monitoring get pods

# Port forward for local access
kubectl -n pulse-monitoring port-forward svc/pulse 8080:80
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
kubectl -n pulse-monitoring scale deployment/pulse --replicas=3

# Or rely on HPA (configured in hpa.yaml)
kubectl -n pulse-monitoring get hpa
```

### Monitoring

```bash
# Check pod logs
kubectl -n pulse-monitoring logs -f deployment/pulse

# Check resource usage
kubectl -n pulse-monitoring top pods

# Check events
kubectl -n pulse-monitoring get events
```

### Upgrades

```bash
# Update image
kubectl -n pulse-monitoring set image deployment/pulse pulse=pulse-monitor/pulse:v1.2.0

# Check rollout status
kubectl -n pulse-monitoring rollout status deployment/pulse

# Rollback if needed
kubectl -n pulse-monitoring rollout undo deployment/pulse
```

### Backup

```bash
# Create backup pod
kubectl -n pulse-monitoring run pulse-backup --rm -i --restart=Never \
  --image=pulse-monitor/pulse:latest \
  --overrides='{"spec": {"containers": [{"name": "backup", "image": "pulse-monitor/pulse:latest", "command": ["pulse-cli", "backup", "-db", "/var/lib/pulse/data/pulse.db", "-output", "/backup/pulse-backup.db"], "volumeMounts": [{"name": "data", "mountPath": "/var/lib/pulse/data"}, {"name": "backup", "mountPath": "/backup"}]}], "volumes": [{"name": "data", "persistentVolumeClaim": {"claimName": "pulse-data"}}, {"name": "backup", "emptyDir": {}}]}}'
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
