# fakturownia

`fakturownia` is an agent-first Go CLI for the [Fakturownia API](https://github.com/fakturownia/API).

It is designed for two audiences at once:

- agents need deterministic behavior, structured output, and stable recovery paths
- humans still need clear help text, sensible defaults, and a clean local workflow

## Status

The current implementation focuses on the first high-value read flows:

- `auth login`
- `auth status`
- `auth logout`
- `invoice list`
- `invoice get`
- `invoice download`
- `schema list`
- `schema <noun> <verb>`
- `doctor run`

The architecture is intentionally structured so more API resources can be added without changing the CLI contract.

## Install

### Download a release

Download the correct archive for macOS or Linux from [GitHub Releases](https://github.com/sixers/fakturownia-cli/releases), extract it, and place `fakturownia` on your `PATH`.

### Build from source

```bash
brew install go
brew install just
git clone https://github.com/sixers/fakturownia-cli.git
cd fakturownia-cli
go build ./cmd/fakturownia
```

## Authentication

The CLI persists API tokens in the OS keychain and stores only profile metadata in the config file.

Supported config inputs:

- `FAKTUROWNIA_API_TOKEN`
- `FAKTUROWNIA_URL`
- `FAKTUROWNIA_PROFILE`

Example:

```bash
fakturownia auth login --prefix acme --api-token "$FAKTUROWNIA_API_TOKEN"
fakturownia auth status --json
```

## Output Contract

Every command supports `--json` or `--output json`.

- JSON is written to stdout
- diagnostics and warnings are written to stderr
- `--raw` emits the upstream JSON response body directly when supported
- `--quiet` emits bare values when exactly one field or column remains

Envelope shape:

```json
{
  "schema_version": "fakturownia-cli/v1alpha1",
  "status": "success",
  "data": {},
  "errors": [],
  "warnings": [],
  "meta": {
    "command": "invoice list",
    "profile": "default",
    "duration_ms": 12
  }
}
```

## Examples

```bash
fakturownia invoice list --json
fakturownia invoice list --period this_month --columns id,number,price_gross
fakturownia invoice get --id 123 --fields id,number,status --json
fakturownia invoice download --id 123 --dir ./invoices --json
fakturownia schema invoice list --json
fakturownia doctor run --json
```

## Exit Codes

- `0` success
- `2` usage or validation error
- `3` not found
- `4` authentication or permission failure
- `5` conflict
- `6` network or timeout failure
- `7` reserved for rate limiting or retry budget exhaustion
- `8` remote API rejected request
- `9` internal CLI failure

## Development

```bash
just test
just lint
just build
```

Golden tests cover help and schema output for the public CLI contract. Run `just schema-help` when you want to refresh just that contract-focused test target.
