# fakturownia

`fakturownia` is an agent-first Go CLI for the [Fakturownia API](https://github.com/fakturownia/API).

It is designed for two audiences at once:

- agents need deterministic behavior, structured output, and stable recovery paths
- humans still need clear help text, sensible defaults, and a clean local workflow

## Supported Commands

The current implementation covers these command groups:

### Auth

- `auth login`
- `auth exchange`
- `auth status`
- `auth logout`

### Accounts

- `account create`
- `account get`
- `account delete`
- `account unlink`

### Departments

- `department list`
- `department get`
- `department create`
- `department update`
- `department delete`
- `department set-logo`

### Issuers

- `issuer list`
- `issuer get`
- `issuer create`
- `issuer update`
- `issuer delete`

### Users

- `user create`

### Categories

- `category list`
- `category get`
- `category create`
- `category update`
- `category delete`

### Clients

- `client list`
- `client get`
- `client create`
- `client update`
- `client delete`

### Payments

- `payment list`
- `payment get`
- `payment create`
- `payment update`
- `payment delete`

### Bank Accounts

- `bank-account list`
- `bank-account get`
- `bank-account create`
- `bank-account update`
- `bank-account delete`

### Products

- `product list`
- `product get`
- `product create`
- `product update`

### Price Lists

- `price-list list`
- `price-list get`
- `price-list create`
- `price-list update`
- `price-list delete`

### Invoices

- `invoice list`
- `invoice get`
- `invoice create`
- `invoice update`
- `invoice delete`
- `invoice send-email`
- `invoice send-gov`
- `invoice change-status`
- `invoice cancel`
- `invoice public-link`
- `invoice add-attachment`
- `invoice download-attachment`
- `invoice download-attachments`
- `invoice fiscal-print`
- `invoice download`

### Recurrings

- `recurring list`
- `recurring create`
- `recurring update`

### Warehouses

- `warehouse list`
- `warehouse get`
- `warehouse create`
- `warehouse update`
- `warehouse delete`

### Warehouse Actions

- `warehouse-action list`

### Warehouse Documents

- `warehouse-document list`
- `warehouse-document get`
- `warehouse-document create`
- `warehouse-document update`
- `warehouse-document delete`

### Webhooks

- `webhook list`
- `webhook get`
- `webhook create`
- `webhook update`
- `webhook delete`

### Schema

- `schema list`
- `schema <noun> <verb>`

### Maintenance

- `self update`

### Diagnostics

- `doctor run`

The architecture is intentionally structured so more API resources can be added without changing the CLI contract.

## Skills

The repo ships a generated, single-installable skill bundle at `skills/fakturownia`.

- root install target: `skills/fakturownia`
- generated bundle-local index: `skills/fakturownia/references/skills-index.md`
- generated recipe index: `skills/fakturownia/recipes/index.md`
- generated repo index for browsing: `docs/skills.md`
- regenerate from code: `just generate-skills`

For GitHub-based skill installers, use repo `sixers/fakturownia-cli` with path `skills/fakturownia`.

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
- can fall back to `gh release download` when `gh` is installed and authenticated

Pin a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/sixers/fakturownia-cli/master/install.sh | VERSION=v0.1.1 bash
```

Install into a custom bin directory:

```bash
curl -fsSL https://raw.githubusercontent.com/sixers/fakturownia-cli/master/install.sh | BIN_DIR=/usr/local/bin bash
```

Run it from a local clone instead of piping from curl:

```bash
./install.sh
VERSION=v0.1.1 ./install.sh
BIN_DIR="$HOME/.local/bin" ./install.sh
```

The `curl ... | bash` path is the recommended install flow for public releases. Running `./install.sh` from a local clone is still handy for development or if you want to inspect the installer before executing it.

### Update an existing install

The recommended update path is the built-in self-update command:

```bash
fakturownia self update
fakturownia --version
```

Preview an update without modifying the binary:

```bash
fakturownia self update --dry-run --json
```

Pin a specific release:

```bash
fakturownia self update --version v0.2.0
fakturownia --version
```

If you are updating from an older release that does not include `self update` yet, rerun the installer script instead:

```bash
curl -fsSL https://raw.githubusercontent.com/sixers/fakturownia-cli/master/install.sh | bash
```

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
  Example: `v0.1.1`
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
fakturownia auth exchange --login user@example.com --password secret --json
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

`schema` describes both the command contract and the README-backed output and request catalogs for supported resources.

- `fakturownia schema auth exchange --json`, `fakturownia schema account get --json`, `fakturownia schema department list --json`, `fakturownia schema department get --json`, `fakturownia schema issuer list --json`, `fakturownia schema issuer get --json`, `fakturownia schema webhook list --json`, `fakturownia schema webhook get --json`, `fakturownia schema invoice list --json`, `fakturownia schema invoice get --json`, `fakturownia schema category list --json`, `fakturownia schema category get --json`, `fakturownia schema client list --json`, `fakturownia schema payment list --json`, `fakturownia schema payment get --json`, `fakturownia schema bank-account list --json`, `fakturownia schema bank-account get --json`, `fakturownia schema product list --json`, `fakturownia schema price-list list --json`, `fakturownia schema price-list get --json`, `fakturownia schema recurring list --json`, `fakturownia schema warehouse list --json`, `fakturownia schema warehouse get --json`, `fakturownia schema warehouse-action list --json`, `fakturownia schema warehouse-document list --json`, and `fakturownia schema warehouse-document get --json` expose `output.known_fields`
- `fakturownia schema account create --json`, `fakturownia schema department create --json`, `fakturownia schema department update --json`, `fakturownia schema issuer create --json`, `fakturownia schema issuer update --json`, `fakturownia schema user create --json`, `fakturownia schema webhook create --json`, `fakturownia schema webhook update --json`, `fakturownia schema invoice create --json`, `fakturownia schema invoice update --json`, `fakturownia schema category create --json`, `fakturownia schema category update --json`, `fakturownia schema client create --json`, `fakturownia schema client update --json`, `fakturownia schema payment create --json`, `fakturownia schema payment update --json`, `fakturownia schema bank-account create --json`, `fakturownia schema bank-account update --json`, `fakturownia schema product create --json`, `fakturownia schema product update --json`, `fakturownia schema price-list create --json`, `fakturownia schema price-list update --json`, `fakturownia schema recurring create --json`, `fakturownia schema recurring update --json`, `fakturownia schema warehouse create --json`, `fakturownia schema warehouse update --json`, and `fakturownia schema warehouse-document create --json`, and `fakturownia schema warehouse-document update --json` expose `request_body_schema`
- `known_fields` and request body catalogs are curated from the upstream [Fakturownia README](https://github.com/fakturownia/API/blob/master/README.md); invoice schemas also cite [KSeF.md](https://github.com/fakturownia/API/blob/master/KSeF.md) for the KSeF-specific `gov_*` fields and payload notes, and invoice plus bank-account schemas cite [API_RACHUNKI_BANKOWE.md](https://github.com/fakturownia/API/blob/master/API_RACHUNKI_BANKOWE.md) for bank-account-specific payloads and invoice bank-account fields
- nested paths use `dot_bracket` syntax such as `positions[].name`
- the catalog is intentionally not exhaustive; syntactically valid paths outside the catalog are still allowed and produce warnings instead of hard failures

Examples:

```bash
fakturownia schema invoice list --json
fakturownia schema auth exchange --json
fakturownia schema account create --json
fakturownia schema department create --json
fakturownia schema issuer create --json
fakturownia schema user create --json
fakturownia schema webhook create --json
fakturownia schema invoice create --json
fakturownia invoice get --id 123 --fields id,number,gov_status,gov_id --json
fakturownia schema recurring create --json
fakturownia schema category create --json
fakturownia schema client create --json
fakturownia schema payment create --json
fakturownia schema bank-account create --json
fakturownia schema product create --json
fakturownia schema price-list create --json
fakturownia schema warehouse create --json
fakturownia schema warehouse-action list --json
fakturownia schema warehouse-document create --json
fakturownia category list --fields name,description --json
fakturownia client list --fields name,email --json
fakturownia payment list --include invoices --fields name,price,paid --json
fakturownia bank-account get --id 100 --fields id,name,bank_account_number,bank_account_version_departments[].show_on_invoice --json
fakturownia product list --fields name,code,stock_level --json
fakturownia price-list get --id 8523 --fields id,name,price_list_positions[].price_gross --json
fakturownia warehouse list --fields name,description --json
fakturownia warehouse-action list --warehouse-document-id 15 --fields kind,product_id,quantity --json
fakturownia warehouse-document get --id 15 --fields id,kind,warehouse_actions[].quantity --json
fakturownia product create --input '{"name":"Widget","code":"W001","tax":"23"}' --json
fakturownia client create --input '{"name":"Acme","email":"billing@example.com"}' --json
fakturownia price-list create --input '{"name":"Dropshipper","currency":"PLN"}' --json
fakturownia warehouse create --input '{"name":"my_warehouse","kind":null,"description":null}' --json
fakturownia warehouse-document create --input '{"kind":"mm","warehouse_actions":[{"product_id":7,"quantity":2,"warehouse2_id":3}]}' --json
fakturownia account create --input '{"account":{"prefix":"acme"},"user":{"login":"owner","email":"owner@example.com","password":"secret"},"company":{"name":"Acme"}}' --json
fakturownia webhook create --input '{"kind":"invoice:create","url":"https://example.com/hook","active":true}' --json
fakturownia invoice list --include-positions --fields number,positions[].name --json
fakturownia invoice create --input '{"kind":"vat","client_id":1,"positions":[{"product_id":1,"quantity":2}]}' --dry-run --json
fakturownia invoice list --columns number,positions[].name
```

## Examples

### Auth

```bash
fakturownia auth login --prefix acme --api-token "$FAKTUROWNIA_API_TOKEN"
fakturownia auth exchange --login user@example.com --password secret --json
fakturownia auth status --json
fakturownia auth logout --yes
```

### Accounts

```bash
fakturownia account create --input '{"account":{"prefix":"acme"},"user":{"login":"owner","email":"owner@example.com","password":"secret"},"company":{"name":"Acme"}}' --json
fakturownia account get --json
fakturownia account unlink --prefix acme --prefix beta --json
fakturownia account delete --yes --dry-run --json
```

### Departments

```bash
fakturownia department list --json
fakturownia department get --id 10 --json
fakturownia department create --input '{"name":"Sales","shortcut":"SALES","tax_no":"1234567890"}' --json
fakturownia department set-logo --id 10 --file ./logo.png --json
```

### Issuers

```bash
fakturownia issuer list --json
fakturownia issuer get --id 3 --json
fakturownia issuer create --input '{"name":"HQ","tax_no":"1234567890"}' --json
fakturownia issuer delete --id 3 --yes --dry-run --json
```

### Users

```bash
fakturownia user create --integration-token PARTNER_TOKEN --input '{"invite":true,"email":"user@example.com","role":"member"}' --json
```

### Clients

```bash
fakturownia client list --json
fakturownia client get --external-id ext-123 --json
fakturownia client create --input '{"name":"Acme"}' --dry-run --json
```

### Categories

```bash
fakturownia category list --json
fakturownia category get --id 100 --json
fakturownia category create --input '{"name":"my_category","description":null}' --dry-run --json
```

### Payments

```bash
fakturownia payment list --include invoices --json
fakturownia payment get --id 555 --json
fakturownia payment create --input '{"name":"Payment 001","price":100.05,"invoice_id":null,"paid":true,"kind":"api"}' --dry-run --json
```

### Bank Accounts

```bash
fakturownia bank-account list --json
fakturownia bank-account get --id 100 --json
fakturownia bank-account create --input '{"name":"Rachunek główny PLN","bank_account_number":"PL61 1090 1014 0000 0712 1981 2874","bank_name":"Santander Bank Polska","bank_currency":"PLN","default":true}' --dry-run --json
fakturownia bank-account update --id 100 --input '{"bank_account_version_departments":[{"department_id":5,"show_on_invoice":true,"main_on_department":true}]}' --json
```

### Products

```bash
fakturownia product list --json
fakturownia product get --id 100 --warehouse-id 7 --json
fakturownia product create --input '{"name":"Widget","code":"W001","price_net":"100","tax":"23"}' --dry-run --json
fakturownia product update --id 333 --input '{"price_gross":"102","tax":"23"}' --json
```

### Price Lists

```bash
fakturownia price-list list --json
fakturownia price-list get --id 8523 --json
fakturownia price-list create --input '{"name":"Dropshipper","currency":"PLN","price_list_positions_attributes":{"0":{"priceable_id":97149307,"price_gross":"33.16","tax":"23"}}}' --dry-run --json
fakturownia price-list update --id 8523 --input '{"description":"updated"}' --json
fakturownia price-list delete --id 8523 --yes --json
```

### Invoices

```bash
fakturownia invoice list --json
fakturownia invoice list --period this_month --columns id,number,price_gross
fakturownia invoice get --id 123 --fields id,number,status --json
fakturownia invoice get --id 123 --fields id,number,gov_status,gov_id,gov_error_messages[] --json
fakturownia invoice get --id 123 --fields id,number,bank_accounts[].bank_name,bank_accounts[].bank_account_number --json
fakturownia invoice get --id 123 --fields number,positions[].name --json
fakturownia invoice get --id 123 --include descriptions --fields descriptions[].content --json
fakturownia invoice get --id 123 --additional-field corrected_content_before --additional-field corrected_content_after --correction-positions full --json
fakturownia invoice create --input '{"kind":"vat","client_id":1,"positions":[{"product_id":1,"quantity":2}]}' --json
fakturownia invoice create --gov-save-and-send --input '{"kind":"vat","buyer_company":true,"seller_tax_no":"5252445767","seller_street":"ul. Przykładowa 10","seller_post_code":"00-001","seller_city":"Warszawa","buyer_name":"Klient ABC Sp. z o.o.","buyer_tax_no":"9876543210","positions":[{"name":"Usługa","quantity":1,"total_price_gross":1230,"tax":23}]}' --json
fakturownia invoice create --input '{"kind":"vat","buyer_name":"Klient ABC","bank_account_id":100,"buyer_mass_payment_code":"ABC-123","positions":[{"name":"Usługa","quantity":1,"total_price_gross":1230,"tax":23}]}' --json
fakturownia invoice update --id 123 --gov-save-and-send --input '{"buyer_name":"Nowa nazwa"}' --json
fakturownia invoice send-email --id 123 --email-to billing@example.com --email-pdf --json
fakturownia invoice send-gov --id 123 --json
fakturownia invoice public-link --id 123 --json
fakturownia invoice add-attachment --id 123 --file ./scan.pdf --json
fakturownia invoice download-attachment --id 123 --kind gov --dir ./attachments --json
fakturownia invoice download-attachment --id 123 --kind gov_upo --dir ./attachments --json
fakturownia invoice fiscal-print --invoice-id 123 --invoice-id 124 --json
fakturownia invoice download --id 123 --dir ./invoices --json
```

For KSeF flows, the API uses `gov` names:

- `invoice send-gov` means “send the invoice to KSeF”
- `invoice download-attachment --kind gov` downloads the KSeF XML
- `invoice download-attachment --kind gov_upo` downloads the KSeF UPO XML
- invoice schemas expose KSeF status through `gov_*` fields such as `gov_status` and `gov_id`

### Recurrings

```bash
fakturownia recurring list --json
fakturownia recurring create --input '{"name":"Miesięczna","invoice_id":1,"every":"1m"}' --json
fakturownia recurring update --id 77 --input '{"next_invoice_date":"2026-05-01"}' --json
```

### Warehouses

```bash
fakturownia warehouse list --json
fakturownia warehouse get --id 1 --json
fakturownia warehouse create --input '{"name":"my_warehouse","kind":null,"description":null}' --json
fakturownia warehouse update --id 1 --input '{"description":"new_description"}' --json
fakturownia warehouse delete --id 1 --yes --json
```

### Warehouse Actions

```bash
fakturownia warehouse-action list --json
fakturownia warehouse-action list --warehouse-id 1 --kind mm --product-id 7 --json
fakturownia warehouse-action list --warehouse-document-id 15 --fields kind,product_id,quantity --json
```

### Warehouse Documents

```bash
fakturownia warehouse-document list --json
fakturownia warehouse-document get --id 15 --json
fakturownia warehouse-document create --input '{"kind":"mm","warehouse_id":1,"warehouse_actions":[{"product_id":7,"quantity":2,"warehouse2_id":3}]}' --json
fakturownia warehouse-document update --id 15 --input '{"invoice_ids":[100,111]}' --json
fakturownia warehouse-document delete --id 15 --yes --json
```

### Webhooks

```bash
fakturownia webhook list --json
fakturownia webhook get --id 7 --json
fakturownia webhook create --input '{"kind":"invoice:create","url":"https://example.com/hook","active":true}' --json
fakturownia webhook update --id 7 --input '{"active":false}' --json
fakturownia webhook delete --id 7 --yes --json
```

### Schema

```bash
fakturownia schema list --json
fakturownia schema auth exchange --json
fakturownia schema account create --json
fakturownia schema department create --json
fakturownia schema issuer create --json
fakturownia schema user create --json
fakturownia schema webhook create --json
fakturownia schema invoice list --json
fakturownia schema invoice create --json
fakturownia schema recurring create --json
fakturownia schema client create --json
fakturownia schema price-list create --json
fakturownia schema warehouse create --json
fakturownia schema warehouse-action list --json
fakturownia schema warehouse-document create --json
```

### Diagnostics

```bash
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
