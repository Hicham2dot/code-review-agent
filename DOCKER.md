# Docker - Code Review Agent

## Build Local

```bash
# Build l'image
docker build -t code-review-agent:latest .

# Vérifier l'image
docker images | grep code-review-agent
```

## Utilisation

### Analyse d'un diff local

```bash
# Analyser un fichier diff
docker run --rm \
  -v $(pwd)/test_vuln.diff:/app/test.diff \
  code-review-agent:latest \
  analyze --file=/app/test.diff --format=cli

# Analyser avec LLM NVIDIA
docker run --rm \
  -e NVIDIA_API_KEY=your_key_here \
  -v $(pwd)/test_vuln.diff:/app/test.diff \
  code-review-agent:latest \
  analyze --file=/app/test.diff --format=json --llm
```

### Docker Compose (avec env vars)

```bash
# Créer fichier .env
echo "NVIDIA_API_KEY=sk-your-key-here" > .env

# Lancer le service
docker-compose up -d

# Exécuter une analyse
docker-compose run code-review-agent analyze --file=/app/diffs/test.diff --format=cli

# Logs
docker-compose logs -f code-review-agent

# Arrêter
docker-compose down
```

## Structure des Volumes

```
Host                          Container
./diffs        ↔  /app/diffs         (diffs à analyser)
./results      ↔  /app/results       (résultats exportés)
```

### Exemple avec volumes

```bash
# Créer dossiers locaux
mkdir -p diffs results

# Copier des diffs
cp test_vuln.diff diffs/

# Lancer et analyser
docker run --rm \
  -e NVIDIA_API_KEY=$NVIDIA_API_KEY \
  -v $(pwd)/diffs:/app/diffs \
  -v $(pwd)/results:/app/results \
  code-review-agent:latest \
  batch --dir=/app/diffs --output=/app/results/report.md --verbose
```

## Variables d'Environnement

| Variable | Requis | Exemple | Notes |
|----------|--------|---------|-------|
| `NVIDIA_API_KEY` | Oui (pour LLM) | `sk-...` | API key NVIDIA |
| `LLM_MODEL` | Non | `google/gemma-3n-e2b-it` | Modèle LLM (défaut: gemma-3n) |
| `CACHE_DIR` | Non | `/app/.cache` | Répertoire cache |
| `CONFIG_FILE` | Non | `/app/config.yml` | Fichier config |

## Production

### Image Minimaliste

```dockerfile
# Build: ~200MB (golang builder + deps)
# Runtime: ~10MB (alpine + ca-certs + binaire)
```

### Security

- ✅ Utilisateur non-root (`reviewer:1000`)
- ✅ Read-only filesystem (si besoin)
- ✅ No secrets in image (via env vars + .env file)
- ✅ Multi-stage build (no build tools in runtime)

### Health Check (optionnel)

```yaml
# docker-compose.yml
services:
  code-review-agent:
    healthcheck:
      test: ["CMD", "./code-review-agent", "--version"]
      interval: 30s
      timeout: 5s
      retries: 3
```

## Troubleshooting

### Permission denied

```bash
# Si l'app ne peut pas écrire dans /app/results
docker run --rm \
  -u 0 \
  -v $(pwd)/results:/app/results \
  code-review-agent:latest \
  analyze --file=/app/test.diff --format=json
```

### NVIDIA_API_KEY not found

```bash
# Vérifier l'env var est passée
docker run --rm -e NVIDIA_API_KEY=$NVIDIA_API_KEY code-review-agent:latest analyze --help
```

### Logs détaillés

```bash
docker run --rm \
  -e NVIDIA_API_KEY=$NVIDIA_API_KEY \
  -v $(pwd)/test.diff:/app/test.diff \
  code-review-agent:latest \
  analyze --file=/app/test.diff --llm --verbose --format=json
```

## Push vers Registry

```bash
# Tag
docker tag code-review-agent:latest myregistry/code-review-agent:1.0

# Push
docker push myregistry/code-review-agent:1.0

# Pull
docker pull myregistry/code-review-agent:1.0
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Build and push Docker image
  uses: docker/build-push-action@v5
  with:
    context: .
    push: true
    tags: myregistry/code-review-agent:${{ github.sha }}
```

### GitLab CI

```yaml
docker_build:
  image: docker:latest
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
```
