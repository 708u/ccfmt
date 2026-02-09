.PHONY: build install test lint

build:
	go build -o ccfmt ./cmd/

install:
	go install ./cmd/

test:
	go test -tags integration ./...

lint:
	golangci-lint run ./...
	golangci-lint fmt ./... && git diff --exit-code
