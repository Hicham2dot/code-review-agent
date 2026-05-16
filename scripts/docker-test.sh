#!/bin/bash

# Docker Build and Test Script
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
IMAGE_NAME="code-review-agent"
IMAGE_TAG="latest"

echo "🐳 Code Review Agent - Docker Test Suite"
echo "========================================"

# 1. Build Image
echo ""
echo "1️⃣  Building Docker image..."
docker build -t $IMAGE_NAME:$IMAGE_TAG $PROJECT_ROOT
if [ $? -eq 0 ]; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi

# 2. Verify Image
echo ""
echo "2️⃣  Verifying image..."
docker images | grep $IMAGE_NAME
echo "✅ Image verified"

# 3. Test Help Command
echo ""
echo "3️⃣  Testing CLI help..."
docker run --rm $IMAGE_NAME:$IMAGE_TAG --help > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ Help command works"
else
    echo "❌ Help command failed"
    exit 1
fi

# 4. Test Version
echo ""
echo "4️⃣  Testing version command..."
docker run --rm $IMAGE_NAME:$IMAGE_TAG version
if [ $? -eq 0 ]; then
    echo "✅ Version command works"
else
    echo "⚠️  Version command not available (OK)"
fi

# 5. Test with sample diff
echo ""
echo "5️⃣  Testing analyze command..."
TEST_DIFF=$(mktemp)
cat > $TEST_DIFF << 'DIFF'
--- a/test.go
+++ b/test.go
@@ -1,3 +1,4 @@
 package main
 
+const apiKey = "sk-1234567890abcdef"
 func main() {}
DIFF

docker run --rm \
    -v $TEST_DIFF:/tmp/test.diff \
    $IMAGE_NAME:$IMAGE_TAG \
    analyze --file=/tmp/test.diff --format=json > /tmp/result.json 2>&1

if [ -f /tmp/result.json ]; then
    echo "✅ Analyze command works"
    echo "Sample output:"
    head -5 /tmp/result.json
else
    echo "⚠️  Analyze did not produce output (check API key)"
fi

rm -f $TEST_DIFF /tmp/result.json

# 6. Docker Compose Test
echo ""
echo "6️⃣  Testing docker-compose configuration..."
docker-compose -f $PROJECT_ROOT/docker-compose.yml config > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ docker-compose.yml is valid"
else
    echo "❌ docker-compose.yml is invalid"
    exit 1
fi

# Summary
echo ""
echo "========================================"
echo "✅ All Docker tests passed!"
echo ""
echo "Next steps:"
echo "  1. Set your NVIDIA_API_KEY in .env"
echo "  2. Run: docker-compose up -d"
echo "  3. Or: docker run --rm -e NVIDIA_API_KEY=\$NVIDIA_API_KEY -v \$(pwd)/diffs:/app/diffs code-review-agent:latest batch --dir=/app/diffs"
