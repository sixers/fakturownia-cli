# AGENTS.md

This file is for agents and contributors making code changes in this repository.

## Purpose

`fakturownia` is an agent-first Go CLI for the [Fakturownia API](https://github.com/fakturownia/API). The goal is not just to wrap HTTP endpoints, but to preserve a stable CLI contract that works well for both humans and automation.

When in doubt, prefer deterministic machine behavior over convenience magic.

## Canonical Sources

- Upstream API contract: `https://github.com/fakturownia/API/blob/master/README.md`
- CLI contract source of truth: `internal/spec/specs.go`
- Command wiring and runtime behavior: `internal/spec/root.go`
- Output, error, and projection behavior: `internal/output/`
- Generated skills source of truth: `internal/spec/skills.go` and `internal/spec/skills_render.go`

For new endpoints and fields, base the implementation on the upstream Fakturownia README, not on live API sampling alone.

## Non-Negotiable CLI Contracts

These are the main invariants of this repo. Do not break them casually.

### 1. Structured output is the primary contract

- Every command supports `--json` or `--output json`.
- JSON goes to stdout.
- Human diagnostics, hints, and warnings go to stderr.
- Successful JSON responses use the common envelope shape from `internal/output/`.
- `--raw` bypasses the envelope and emits the upstream JSON response body directly when supported.

### 2. Exit codes matter

Keep the semantic exit code scheme stable:

- `0` success
- `2` usage or validation error
- `3` not found
- `4` authentication or permission failure
- `5` conflict
- `6` network or timeout failure
- `7` reserved for throttling or retry budget exhaustion
- `8` remote API rejected request
- `9` internal CLI failure

### 3. Preserve the noun -> verb command grammar

Top-level command groups are nouns such as `auth`, `client`, `invoice`, `schema`, `doctor`, and `self`.

When adding a new API area, prefer a singular noun for CLI consistency unless there is a very strong reason not to.

### 4. Preserve upstream field names

- Pass-through API payloads should keep upstream field names.
- Do not invent a normalized domain model for invoices, clients, or future resources.
- Use README-backed catalogs in `internal/spec/` for discoverability, not for rewriting the payload shape.

### 5. Keep `--fields` and `--columns` separate

- `--fields` limits JSON `data`.
- `--columns` controls human table rendering only.
- Nested selectors use `dot_bracket` syntax such as `positions[].name`.
- Valid but undocumented field paths should warn, not fail, when they are syntactically valid.

### 6. `--dry-run` is part of the public contract

- Read-only commands accept it and ignore it.
- Mutating commands should validate input, auth resolution, and request shape, then return a request plan without sending the request.

## Repository Layout

- `cmd/fakturownia/`: CLI entrypoint
- `cmd/gen-skills/`: generator for installable skills docs
- `internal/auth/`: auth workflows and keychain integration
- `internal/config/`: config resolution and env/profile precedence
- `internal/transport/`: HTTP transport, retries, request planning, raw body capture
- `internal/output/`: envelopes, errors, projection, nested path parsing, table rendering
- `internal/<resource>/`: resource services such as `client`, `invoice`, `doctor`, `selfupdate`
- `internal/spec/`: command registry, schema generation, output catalogs, request body specs, skill metadata
- `docs/skills.md`: generated repo browsing index
- `skills/fakturownia/`: generated installable skill bundle
- `testdata/golden/`: contract snapshots for help and schema output

## Working Rules

### Generated files

Do not hand-edit generated skill docs.

These are generated from code:

- `docs/skills.md`
- `skills/fakturownia/SKILL.md`
- `skills/fakturownia/references/skills-index.md`
- `skills/fakturownia/subskills/**`

If you change skill metadata or command examples, update the source in `internal/spec/skills.go` or `internal/spec/skills_render.go`, then run:

```bash
just generate-skills
go run ./cmd/gen-skills --check
```

### Golden files

Help and schema output are contract-tested via goldens in `testdata/golden/`.

If you intentionally change help or schema output, update the relevant goldens:

```bash
UPDATE_GOLDEN=1 go test ./internal/spec -run TestGolden -count=1
```

Then rerun the normal tests.

### HTTP and JSON behavior

- Use `internal/transport.Client` for API access.
- Read-only endpoints should use idempotent GETs and can use retries.
- Mutating endpoints should not automatically retry writes.
- Preserve `json.Number` behavior from transport; do not reintroduce float formatting bugs by decoding through `float64`.
- For mutating commands, use `transport.PlanJSONRequest(...)` for dry-run output.

### Schema and discovery behavior

- Every new command must be represented in `internal/spec/specs.go`.
- If the command returns passthrough upstream objects, add an `OutputSpec` and, where helpful, a README-backed output catalog.
- If the command accepts structured JSON input, add a `RequestBodySpec` and `request_body_schema`.
- `schema` should stay authoritative for command contracts, output discovery, and request discovery.

## Adding a New Resource or Endpoint

Use this checklist when implementing a new API area such as `product`, `payment`, `warehouse`, or additional verbs under an existing noun.

### 1. Implement the service layer

Create or extend `internal/<resource>/service.go`.

Typical patterns:

- request structs contain `ConfigPath`, `Profile`, `Env`, `Timeout`, and `MaxRetries`
- response structs include `RawBody`, `Profile`, `RequestID`, and any CLI-generated metadata
- list responses synthesize pagination with:
  - `page`
  - `per_page`
  - `returned`
  - `has_next = returned == per_page`

For passthrough JSON resources, return `map[string]any` or `[]map[string]any`.

### 2. Wire the service into the binary

Update `cmd/fakturownia/main.go` and `internal/spec/root.go`:

- add the dependency interface
- add the concrete service to `Dependencies`
- add the noun command and subcommands

### 3. Add the command registry entry

Update `internal/spec/specs.go` for every new command:

- `Noun`
- `Verb`
- `Use`
- `Short`
- `Examples`
- `EnvVars`
- `LocalFlags`
- `OutputModes`
- `ExitCodes`
- `RawSupported`
- `Mutating`
- `DataPrototype`
- `Output`
- `RequestBody`

The registry is the source of truth for:

- help text
- schema output
- skill examples
- command discovery

### 4. Keep root command behavior aligned

In `internal/spec/root.go`:

- validate required and mutually exclusive flags
- enforce `--yes` where needed
- forward config/env/profile/timeout/retry inputs
- use `prepareOutputOptions(...)`
- return human output via `output.RenderSuccess(...)`
- return machine errors via `writeCommandError(...)`

### 5. Add output catalogs for discoverability

If the response is an upstream object or array, add a README-backed output catalog in `internal/spec/`.

Follow the existing invoice/client pattern:

- keep schemas open-ended
- mark known fields as curated, not exhaustive
- only mark nested paths as projectable if runtime projection actually supports them

### 6. Add request-body specs for writes

For mutating commands that accept JSON input:

- add `RequestBodySpec` in `internal/spec/request_body.go` or a related file
- expose `request_body_schema`
- accept the inner object only
- let the CLI wrap it into the upstream envelope

### 7. Update skills if the command surface changed

If you add a new noun or materially expand an existing area:

- update `internal/spec/skills.go`
- add command refs, discovery hints, and examples
- run `just generate-skills`

If you add a brand-new top-level API area, add a dedicated subskill rather than burying it in `shared`.

### 8. Add tests

At minimum:

- service tests in `internal/<resource>/service_test.go`
- command integration or golden coverage in `internal/spec/root_test.go`
- output/path behavior tests if nested fields or rendering changed
- skill generation tests if a new skill area or metadata changed

### 9. Update README when the user-facing surface changed

Update `README.md` for:

- new supported commands
- new flags or workflows
- new examples
- changed installation, maintenance, or diagnostics guidance

## Resource Implementation Patterns

### Read-only list/get endpoints

- default `--per-page` to `25`
- validate `1 <= per-page <= 100`
- synthesize pagination locally because Fakturownia list endpoints return bare arrays
- expose `--raw` when the upstream response is JSON

### Mutating endpoints

- support `--dry-run`
- prefer `--input -|@file|JSON` for structured bodies
- use request plans in JSON output for dry-run mode
- require explicit `--yes` for destructive operations

### Error handling

Prefer `internal/output` helpers:

- `output.Usage(...)`
- `output.NotFound(...)`
- `output.Conflict(...)`
- `output.Network(...)`
- `output.Remote(...)`
- `output.Internal(...)`

Always provide a machine-readable code and a bounded hint.

## Skills and Agent Discovery

This repo ships a single installable skill bundle at `skills/fakturownia`.

Keep these principles when changing it:

- skill content is generated from code, not hand-maintained markdown
- the root skill should stay short and point agents to the index and `shared`
- area skills should describe when to use them, which commands they cover, and which schema/discovery flows matter
- `shared` is for global CLI behavior, not a dumping ground for unrelated API docs

## Development Commands

Use these before finishing a change:

```bash
just test
just lint
just build
go run ./cmd/gen-skills --check
```

Useful extras:

```bash
just generate-skills
UPDATE_GOLDEN=1 go test ./internal/spec -run TestGolden -count=1
```

## Release Notes

If asked to cut a release:

- the default branch is `master`
- push the release commit to `origin/master`
- create and push a semver tag such as `v0.3.0`
- GitHub Actions handles packaging and publishing

Current release artifacts target:

- `darwin`
- `linux`
- `amd64`
- `arm64`

Windows is intentionally not part of the current release matrix.

## Practical Advice

- Prefer additive changes to the public contract.
- Keep examples realistic and copy-pasteable.
- If you change contract text, expect golden files to need updates.
- If you change command metadata, expect skills output to need regeneration.
- If you touch payload projection or rendering, test both JSON and human output.
- If you are implementing a new endpoint family, follow the existing `client` and `invoice` patterns instead of inventing a new architecture.
