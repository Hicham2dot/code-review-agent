#!/bin/bash

# Load environment variables from .env if it exists
if [ -f .env ]; then
    export $(cat .env | grep -v '#' | xargs)
fi

# Check if MISTRAL_API_KEY is set
if [ -z "$MISTRAL_API_KEY" ]; then
    echo "❌ Error: MISTRAL_API_KEY is not set"
    echo "Please set it in .env file or as environment variable"
    exit 1
fi

echo "✅ MISTRAL_API_KEY is configured"
echo "Running: $@"
exec "$@"
