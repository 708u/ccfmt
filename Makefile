.PHONY: build install test

build:
	go build -o ccfmt ./cmd/

install:
	go install ./cmd/

test:
	go test -tags integration ./...
