.PHONY: test run attach

test:
	docker build --no-cache -f Dockerfile.test . -t store_test

run:
	docker build --no-cache -f Dockerfile . -t store
	docker run --name store_run -it --rm store

attach:
	docker exec store_run tail -f /app/agent.log

