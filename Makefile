.PHONY: test run

test:
	docker build -f Dockerfile.test . -t store_test

run:
	docker build -f Dockerfile . -t store
	docker run -it --rm store

