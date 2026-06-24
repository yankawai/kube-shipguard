SHELL := /bin/sh

BINARY := kube-shipguard
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: build
build:
	mkdir -p bin
	go build -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o bin/$(BINARY) ./cmd/kube-shipguard

.PHONY: scan
scan:
	go run ./cmd/kube-shipguard scan examples/secure --format text --fail-on high

.PHONY: demo
demo:
	go run ./cmd/kube-shipguard scan examples/unsafe --format text --fail-on none

.PHONY: sarif
sarif:
	go run ./cmd/kube-shipguard scan examples/unsafe --format sarif --output kube-shipguard.sarif --fail-on none

.PHONY: validate
validate: test vet scan
