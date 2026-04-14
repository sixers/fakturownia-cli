SHELL := /bin/bash

GO ?= go
STATICCHECK ?= staticcheck

.PHONY: test lint build schema-help

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	$(STATICCHECK) ./...

build:
	$(GO) build ./cmd/fakturownia

schema-help:
	$(GO) test ./internal/spec -run 'TestGolden' -count=1
