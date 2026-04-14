# Contributing

## Local setup

```bash
brew install go
brew install just
just test
```

## Principles

- preserve the agent-first contract
- keep JSON output stable and deterministic
- route all command metadata through the `CommandSpec` registry
- prefer additive changes to the command surface

## Before opening a change

Run:

```bash
just test
just lint
```

If your change affects help or schema output, update the related golden files.
