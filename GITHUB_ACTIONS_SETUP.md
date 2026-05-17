# GitHub Actions - Security Check Setup

## 🎯 Objectif

Analyser automatiquement tous les commits et pull requests pour détecter les vulnérabilités. Les vulnérabilités **critiques bloquent le merge**.

## ⚙️ Configuration

### 1. Ajouter votre clé NVIDIA API aux secrets GitHub

1. Allez dans **Settings** → **Secrets and variables** → **Actions**
2. Cliquez **New repository secret**
3. Nom: `NVIDIA_API_KEY`
4. Valeur: Votre clé API NVIDIA (obtenue sur https://build.nvidia.com)
5. **Save**

### 2. Fichier workflow (déjà créé)

Le fichier `.github/workflows/security-check.yml` contient :

```yaml
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]
```

**Se déclenche sur** :
- ✅ Tous les push sur `main` ou `develop`
- ✅ Tous les PR vers `main` ou `develop`

### 3. Étapes du workflow

1. **Checkout** : Récupère le code
2. **Build Docker** : Compile l'image `code-review-agent`
3. **Extract diff** : Génère le diff des modifications
4. **Run Analysis** : Exécute l'analyse (local + LLM NVIDIA)
5. **Comment PR** : Poste les résultats comme commentaire sur la PR
6. **Check criticals** : Bloque le merge si vulnérabilités critiques trouvées

## 📊 Résultats

### Sur une Pull Request

Le workflow affiche un **commentaire automatique** :

```
## 🔍 Code Review Analysis

Status: ❌ FAILED

Summary: 8 issue(s) found
- Critical: 3
- Major: 2
- Minor: 3

### Issues

1. 🔴 hardcoded_secrets (critical)
   - File: database.go:5
   - Message: Hardcoded api_key detected in code
   - Suggestion: Use environment variables or secret management service
   - Confidence: 98%
```

### Blocage du merge

Si **1+ vulnérabilité critique** est détectée :
- ❌ Status de la PR devient `FAILED`
- 🚫 Merge est **bloqué** jusqu'à résolution

## 🔑 Variables d'environnement

| Variable | Valeur | Source |
|----------|--------|--------|
| `NVIDIA_API_KEY` | Votre clé API | GitHub Secrets |
| `REVIEW_LLM_MODEL` | google/gemma-3n-e2b-it | docker-compose.yml (optionnel) |

## 🧪 Test du workflow

### Créer une PR de test

```bash
# Créer une branche
git checkout -b test/vulnerable-code

# Ajouter du code vulnérable
cat > vulnerable.go << 'EOF'
package main

const apiKey = "sk-1234567890abcdef"

func QueryDB(userInput string) {
    query := "SELECT * FROM users WHERE id = " + userInput
    db.Exec(query)
}
EOF

# Commit et push
git add vulnerable.go
git commit -m "test: add vulnerable code for CI testing"
git push origin test/vulnerable-code

# Créer PR sur GitHub
# → Le workflow se déclenche automatiquement
# → Les résultats s'affichent dans les commentaires
# → Le merge est bloqué si vulnérabilités critiques
```

## 📝 Personnalisations possibles

### Changer les branches

Modifiez `.github/workflows/security-check.yml` :

```yaml
on:
  push:
    branches: [ main, staging, production ]
  pull_request:
    branches: [ main, staging, production ]
```

### Avertir au lieu de bloquer

Changez l'exit code final :

```yaml
- name: Check for critical issues
  run: |
    if [ -f /tmp/output/result.json ]; then
      CRITICAL_COUNT=$(jq '[.issues[] | select(.severity=="critical")] | length' /tmp/output/result.json)
      if [ "$CRITICAL_COUNT" -gt 0 ]; then
        echo "⚠️ Found $CRITICAL_COUNT critical issue(s) - warning only"
        # Ne pas exit 1 pour ne pas bloquer
      fi
    fi
```

### Envoyer les résultats en Slack

Ajoutez un step :

```yaml
- name: Notify Slack
  if: failure()
  uses: slackapi/slack-github-action@v1.24.0
  with:
    webhook-url: ${{ secrets.SLACK_WEBHOOK }}
    payload: |
      {
        "text": "🚨 Security check failed on ${{ github.ref }}"
      }
```

## ✅ Checklist de configuration

- [ ] NVIDIA_API_KEY ajoutée dans GitHub Secrets
- [ ] `.github/workflows/security-check.yml` commité et pushé
- [ ] Branche protégée : Settings → Branches → Add rule → Require status checks to pass
- [ ] Test avec une PR vulnérable
- [ ] Vérifier que les commentaires apparaissent automatiquement

## 🆘 Troubleshooting

### Workflow ne se déclenche pas

→ Vérifiez que le fichier est dans `.github/workflows/` et commité

### Erreur "NVIDIA_API_KEY not found"

→ Allez dans Settings → Secrets et ajoutez la clé

### Analyse dit "0 fichiers" ou "vide"

→ Vérifiez que `git diff` génère bien les modifications

### Le merge n'est pas bloqué

→ Allez dans Settings → Branches → Require status checks
