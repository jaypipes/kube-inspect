VERSION ?= $(shell git describe --tags --always --dirty)

.PHONY: test build

build:
	@go build ./...

test:
	@go test -coverprofile=cover.out -v ./...

cover-html: test
	@go tool cover -html=cover.out -o cover.html
	@echo "open cover.html with a browser"

cover: test
	@go tool cover -func=cover.out
