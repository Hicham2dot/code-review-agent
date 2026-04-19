#!/bin/bash
set -e

echo "Running tests..."
go test ./...
echo "All tests passed"
