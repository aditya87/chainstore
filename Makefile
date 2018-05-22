.PHONY: build test

build:
	@echo "Building Docker image"
	docker build -f Dockerfile . -t agent_test

test: build
	docker run --rm agent_test

run: build
	docker run --rm -e PORT=3000 -e REDIS_PORT=7777 -it agent_test /bin/bash
