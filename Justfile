set shell := ["bash", "-cu"]

go := env_var_or_default("GO", "go")
staticcheck := env_var_or_default("STATICCHECK", "$(go env GOPATH)/bin/staticcheck")

default:
  @just --list

test:
  {{go}} test ./...

lint:
  {{go}} vet ./...
  {{staticcheck}} ./...

build:
  {{go}} build ./cmd/fakturownia

schema-help:
  {{go}} test ./internal/spec -run 'TestGolden' -count=1
