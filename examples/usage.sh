#!/bin/bash

# Local analysis
./code-review-agent analyze --file=example.diff

# With LLM
./code-review-agent analyze --file=example.diff --llm=claude

# Batch mode
./code-review-agent batch --repo=.

# Clear cache
./code-review-agent cache-clear
