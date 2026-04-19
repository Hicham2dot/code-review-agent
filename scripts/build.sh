#!/bin/bash
set -e

echo "Building binary..."
go build -o code-review-agent ./cmd
echo "Binary built: ./code-review-agent"
