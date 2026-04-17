set shell := ["bash", "-cu"]

go := env_var_or_default("GO", "go")
staticcheck := env_var_or_default("STATICCHECK", "$(go env GOPATH)/bin/staticcheck")
gitleaks := env_var_or_default("GITLEAKS", "$(go env GOPATH)/bin/gitleaks")

default:
  @just --list

test:
  {{go}} test ./...

lint:
  {{go}} vet ./...
  {{staticcheck}} ./...

build:
  {{go}} build ./cmd/fakturownia

secrets:
  {{gitleaks}} detect --no-banner --source . --config gitleaks.toml

generate-skills:
  {{go}} run ./cmd/gen-skills

schema-help:
  {{go}} test ./internal/spec -run 'TestGolden' -count=1
