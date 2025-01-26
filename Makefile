VERSION ?= $(shell git describe --tags --always --dirty)

.PHONY: test build

build:
	@go build ./...

test:
	@go test -cover -v ./...
