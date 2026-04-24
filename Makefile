.PHONY: help build test run docker-build clean

help:
	@echo "Available commands:"
	@echo "  make build       - Build the binary"
	@echo "  make test        - Run tests"
	@echo "  make run         - Run the agent"
	@echo "  make docker-build - Build Docker image"
	@echo "  make clean       - Clean build artifacts"

build:
	bash scripts/build.sh

test:
	bash scripts/test.sh

run: build
	./code-review-agent

docker-build:
	bash scripts/docker_build.sh

clean:
	rm -f code-review-agent
	rm -rf .cache/
