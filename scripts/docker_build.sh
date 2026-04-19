#!/bin/bash
set -e

echo "Building Docker image..."
docker build -t code-review-agent:latest .
echo "Docker image built"
