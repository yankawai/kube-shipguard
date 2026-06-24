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
	go run ./cmd/kube-shipguard scan examples --format text --fail-on high

.PHONY: sarif
sarif:
	go run ./cmd/kube-shipguard scan examples --format sarif --output kube-shipguard.sarif --fail-on high

.PHONY: validate
validate: test vet scan
