.PHONY: build test

build:
	@echo "Building Docker image"
	docker build -f Dockerfile . -t agent_test

test: build
	docker run agent_test
