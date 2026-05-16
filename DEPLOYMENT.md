# Deployment Guide - Code Review Agent

## Environment Setup

### 1. Prerequisites
- Docker & Docker Compose installed
- NVIDIA API Key (for LLM analysis) - https://build.nvidia.com
- Git for version control

### 2. Environment Configuration

```bash
# Copy example
cp .env.example .env

# Edit .env with your NVIDIA API Key
export NVIDIA_API_KEY=sk-your-key-here
```

## Local Development

### Build and Test
```bash
# Build Docker image
make -f Makefile.docker docker-build

# Run tests
make -f Makefile.docker docker-test

# Test with sample diff
make -f Makefile.docker docker-analyze
```

## Docker Deployment

### Single Container

```bash
# Build
docker build -t code-review-agent:latest .

# Run analysis
docker run --rm \
  -e NVIDIA_API_KEY=$NVIDIA_API_KEY \
  -v $(pwd)/diffs:/app/diffs \
  -v $(pwd)/results:/app/results \
  code-review-agent:latest \
  batch --dir=/app/diffs --output=/app/results/report.md
```

### Docker Compose

```bash
# Start service
docker-compose up -d

# Check logs
docker-compose logs -f code-review-agent

# Stop
docker-compose down
```

## Production Checklist

- [ ] NVIDIA_API_KEY configured securely (secrets manager)
- [ ] Docker image built and tested locally
- [ ] Image pushed to private registry
- [ ] Volumes mounted to persistent storage
- [ ] Logging aggregated (ELK, CloudWatch, etc.)
- [ ] Health checks configured
- [ ] Resource limits set (CPU, Memory)
- [ ] Non-root user (reviewer:1000)
- [ ] Read-only root filesystem (optional)
- [ ] Network policies configured

## Container Registry

### Build and Push

```bash
# AWS ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 123456789.dkr.ecr.us-east-1.amazonaws.com
docker build -t 123456789.dkr.ecr.us-east-1.amazonaws.com/code-review-agent:1.0 .
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/code-review-agent:1.0

# Docker Hub
docker tag code-review-agent:latest myuser/code-review-agent:1.0
docker push myuser/code-review-agent:1.0

# GitLab Registry
docker tag code-review-agent:latest registry.gitlab.com/mygroup/myproject:1.0
docker push registry.gitlab.com/mygroup/myproject:1.0
```

## Kubernetes Deployment (Optional)

### Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: code-review-agent
spec:
  replicas: 1
  selector:
    matchLabels:
      app: code-review-agent
  template:
    metadata:
      labels:
        app: code-review-agent
    spec:
      containers:
      - name: code-review-agent
        image: code-review-agent:latest
        env:
        - name: NVIDIA_API_KEY
          valueFrom:
            secretKeyRef:
              name: nvidia-secret
              key: api-key
        volumeMounts:
        - name: diffs
          mountPath: /app/diffs
        - name: results
          mountPath: /app/results
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: diffs
        persistentVolumeClaim:
          claimName: diffs-pvc
      - name: results
        persistentVolumeClaim:
          claimName: results-pvc
```

### Create Secret
```bash
kubectl create secret generic nvidia-secret --from-literal=api-key=$NVIDIA_API_KEY
```

## Monitoring & Logging

### Container Logs
```bash
# Local
docker-compose logs -f code-review-agent

# Kubernetes
kubectl logs -f deployment/code-review-agent
```

### Health Checks
```bash
# Test endpoint
docker exec <container> ./code-review-agent --version

# Check exit status
echo $?  # 0 = healthy
```

## Scaling

### Horizontal Scaling (Batch Processing)
```bash
# Process multiple diffs in parallel
docker run --rm \
  -e NVIDIA_API_KEY=$NVIDIA_API_KEY \
  -v $(pwd)/diffs:/app/diffs \
  code-review-agent:latest \
  batch --dir=/app/diffs --workers=4
```

### Resource Limits (Docker)
```yaml
# docker-compose.yml
services:
  code-review-agent:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Troubleshooting

### Image Size Optimization
```bash
# Check image size
docker images code-review-agent

# Expected: ~10-15MB (runtime) + binary
```

### API Rate Limiting
If NVIDIA API returns rate limit errors:
- Implement exponential backoff
- Add retry logic in analyzer
- Use batch processing with delays

### Memory Issues
```bash
# Monitor container memory
docker stats code-review-agent

# Increase limit if needed
docker run --memory=1g code-review-agent:latest
```

## Rollback Strategy

```bash
# Keep previous image tagged
docker tag code-review-agent:latest code-review-agent:v1.0
docker tag code-review-agent:v1.0 code-review-agent:previous

# Rollback
docker run code-review-agent:previous
```

## Cleanup

```bash
# Remove old images
docker image prune -a

# Remove dangling volumes
docker volume prune

# Full cleanup
docker system prune -a
```
