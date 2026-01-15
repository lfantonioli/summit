.PHONY: build test integration-test format

build:
	CGO_ENABLED=0 go build

test:
	go test -count=1 ./...

integration-test:
	./scripts/run-integration-tests.sh

format:
	go fmt ./...
