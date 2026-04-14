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

### Install with the script

The public install path is:

```bash
curl -fsSL https://raw.githubusercontent.com/sixers/fakturownia-cli/master/install.sh | bash
```

The installer:

- detects `darwin` or `linux`
- detects `amd64` or `arm64`
- downloads the matching release archive
- verifies it against `checksums.txt`
- installs `fakturownia` into `~/.local/bin` by default
- prints a PATH hint if `~/.local/bin` is not already on your PATH
- uses `GITHUB_TOKEN` or `GH_TOKEN` when set
- falls back to `gh release download` when `gh` is installed and authenticated

Pin a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/sixers/fakturownia-cli/master/install.sh | VERSION=v0.1.0 bash
```

Install into a custom bin directory:

```bash
curl -fsSL https://raw.githubusercontent.com/sixers/fakturownia-cli/master/install.sh | BIN_DIR=/usr/local/bin bash
```

Run it from a local clone instead of piping from curl:

```bash
./install.sh
VERSION=v0.1.0 ./install.sh
BIN_DIR="$HOME/.local/bin" ./install.sh
```

While the repo is still private, the local-clone path is the practical option. If `gh` is installed and authenticated, the script can still download private release assets. Once the repo is public, the `curl ... | bash` flow works as-is.

### Build from source

This is the simplest copy-paste path during early development:

```bash
brew install go just
git clone https://github.com/sixers/fakturownia-cli.git
cd fakturownia-cli
mkdir -p "$HOME/.local/bin"
go build -o "$HOME/.local/bin/fakturownia" ./cmd/fakturownia
case "$(basename "$SHELL")" in
  zsh) rc_file="$HOME/.zshrc" ;;
  bash) rc_file="$HOME/.bashrc" ;;
  *) rc_file="$HOME/.profile" ;;
esac
grep -qxF 'export PATH="$HOME/.local/bin:$PATH"' "$rc_file" || echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$rc_file"
export PATH="$HOME/.local/bin:$PATH"
fakturownia --version
```

### Install from a release

```bash
mkdir -p "$HOME/.local/bin"
tmpdir="$(mktemp -d)"
cd "$tmpdir"
curl -fsSLO "https://github.com/sixers/fakturownia-cli/releases/download/VERSION/fakturownia_VERSION_OS_ARCH.tar.gz"
tar -xzf "fakturownia_VERSION_OS_ARCH.tar.gz"
install -m 0755 fakturownia "$HOME/.local/bin/fakturownia"
rm -rf "$tmpdir"
case "$(basename "$SHELL")" in
  zsh) rc_file="$HOME/.zshrc" ;;
  bash) rc_file="$HOME/.bashrc" ;;
  *) rc_file="$HOME/.profile" ;;
esac
grep -qxF 'export PATH="$HOME/.local/bin:$PATH"' "$rc_file" || echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$rc_file"
export PATH="$HOME/.local/bin:$PATH"
fakturownia --version
```

Replace:

- `VERSION` with a release tag such as `v0.1.0`
- `OS` with `darwin` or `linux`
- `ARCH` with `amd64` or `arm64`

The manual install path is mostly useful for debugging or air-gapped installs. The script above is the recommended path for normal users.

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

## Output Introspection

`schema` describes both the command contract and the README-backed output catalog for invoice resources.

- `fakturownia schema invoice list --json` exposes `output.known_fields`
- `known_fields` is curated from the upstream [Fakturownia README](https://github.com/fakturownia/API/blob/master/README.md)
- nested paths use `dot_bracket` syntax such as `positions[].name`
- the catalog is intentionally not exhaustive; syntactically valid paths outside the catalog are still allowed and produce warnings instead of hard failures

Examples:

```bash
fakturownia schema invoice list --json
fakturownia invoice list --include-positions --fields number,positions[].name --json
fakturownia invoice list --columns number,positions[].name
```

## Examples

```bash
fakturownia invoice list --json
fakturownia invoice list --period this_month --columns id,number,price_gross
fakturownia invoice get --id 123 --fields id,number,status --json
fakturownia invoice get --id 123 --fields number,positions[].name --json
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

## Release

Releases are created by pushing a semver tag. The GitHub Actions `release` workflow then runs GoReleaser and publishes the archives, checksums, and SBOMs automatically.

Prerequisites:

- push the release commit to `master`
- have `gh` authenticated for the `sixers/fakturownia-cli` repo
- use a clean worktree before tagging

Dry run locally:

```bash
brew install goreleaser
goreleaser release --snapshot --clean
```

Create a real release:

```bash
cd /Users/mateusz/Projects/Personal/fakturownia-cli

just test
just lint
just build

git status --short
git push origin master

version="v0.1.0"
git tag "$version"
git push origin "$version"
```

Watch the release workflow:

```bash
gh run list --repo sixers/fakturownia-cli --workflow release --limit 5
run_id="$(gh run list --repo sixers/fakturownia-cli --workflow release --limit 1 --json databaseId --jq '.[0].databaseId')"
gh run watch "$run_id" --repo sixers/fakturownia-cli
gh release view "$version" --repo sixers/fakturownia-cli
```

If you need to replace a failed tag before publishing a corrected release:

```bash
version="v0.1.0"
git tag -d "$version"
git push origin ":refs/tags/$version"
git tag "$version"
git push origin "$version"
```
