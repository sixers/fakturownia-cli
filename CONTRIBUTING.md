# Contributing

## Local setup

```bash
brew install go
go test ./...
```

## Principles

- preserve the agent-first contract
- keep JSON output stable and deterministic
- route all command metadata through the `CommandSpec` registry
- prefer additive changes to the command surface

## Before opening a change

Run:

```bash
go test ./...
go vet ./...
staticcheck ./...
```

If your change affects help or schema output, update the related golden files.
