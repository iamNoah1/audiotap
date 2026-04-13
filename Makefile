MODULE  := github.com/iamNoah1/audiotap
BINARY  := audiotap
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X $(MODULE)/cmd.version=$(VERSION)

.PHONY: build test vet install clean

## build: compile the binary for the current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: run all tests
test:
	go test -race -timeout 120s ./...

## vet: run go vet
vet:
	go vet ./...

## install: install binary to $GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" .

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/

.DEFAULT_GOAL := build

help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
