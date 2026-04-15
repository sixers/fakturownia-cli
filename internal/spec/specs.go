package spec

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/invopop/jsonschema"

	"github.com/sixers/fakturownia-cli/internal/account"
	"github.com/sixers/fakturownia-cli/internal/auth"
	"github.com/sixers/fakturownia-cli/internal/bankaccount"
	"github.com/sixers/fakturownia-cli/internal/category"
	"github.com/sixers/fakturownia-cli/internal/client"
	"github.com/sixers/fakturownia-cli/internal/department"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/issuer"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/payment"
	"github.com/sixers/fakturownia-cli/internal/pricelist"
	"github.com/sixers/fakturownia-cli/internal/selfupdate"
	"github.com/sixers/fakturownia-cli/internal/warehouse"
	"github.com/sixers/fakturownia-cli/internal/warehousedocument"
	"github.com/sixers/fakturownia-cli/internal/webhook"
)

type FlagSpec struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Default     string   `json:"default,omitempty"`
	Repeatable  bool     `json:"repeatable,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type EnvVarSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ExitCodeSpec struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

type CommandSpec struct {
	Noun          string
	Verb          string
	Use           string
	Short         string
	Examples      []string
	EnvVars       []EnvVarSpec
	LocalFlags    []FlagSpec
	OutputModes   []string
	ExitCodes     []ExitCodeSpec
	RawSupported  bool
	Mutating      bool
	DataPrototype any
	Output        *OutputSpec
	RequestBody   *RequestBodySpec
}

type SchemaSummary struct {
	Noun    string `json:"noun"`
	Verb    string `json:"verb"`
	Use     string `json:"use"`
	Summary string `json:"summary"`
}

type CommandSchema struct {
	SchemaVersion     string           `json:"schema_version"`
	Command           string           `json:"command"`
	Use               string           `json:"use"`
	Summary           string           `json:"summary"`
	Flags             []FlagSpec       `json:"flags"`
	EnvVars           []EnvVarSpec     `json:"env_vars"`
	OutputModes       []string         `json:"output_modes"`
	ExitCodes         []ExitCodeSpec   `json:"exit_codes"`
	RawSupported      bool             `json:"raw_supported"`
	Examples          []string         `json:"examples"`
	DataSchema        map[string]any   `json:"data_schema"`
	RequestBody       *RequestBodySpec `json:"request_body,omitempty"`
	RequestBodySchema map[string]any   `json:"request_body_schema,omitempty"`
	EnvelopeSchema    map[string]any   `json:"envelope_schema"`
	Output            *OutputSpec      `json:"output,omitempty"`
}

func GlobalFlags() []FlagSpec {
	return []FlagSpec{
		{Name: "profile", Type: "string", Description: "Select a named profile", Default: ""},
		{Name: "json", Type: "bool", Description: "Alias for --output json", Default: "false"},
		{Name: "output", Type: "string", Description: "Output format", Default: "human", Enum: []string{"human", "json"}},
		{Name: "quiet", Type: "bool", Description: "Emit bare values when exactly one field or column remains", Default: "false"},
		{Name: "fields", Type: "string[]", Description: "Project JSON envelope data fields using dot/bracket paths like number or positions[].name", Repeatable: true},
		{Name: "columns", Type: "string[]", Description: "Select human table columns using dot/bracket paths like number or positions[].name", Repeatable: true},
		{Name: "raw", Type: "bool", Description: "Emit the upstream JSON response body directly when supported", Default: "false"},
		{Name: "dry-run", Type: "bool", Description: "Accepted on read-only commands and reserved for future mutating request previews", Default: "false"},
		{Name: "timeout-ms", Type: "int", Description: "HTTP timeout in milliseconds", Default: "30000"},
		{Name: "max-retries", Type: "int", Description: "Maximum retry attempts for idempotent reads on network or 5xx failures", Default: "2"},
		{Name: "non-interactive", Type: "bool", Description: "Disable interactive behavior", Default: "true"},
		{Name: "config", Type: "string", Description: "Override the config file path", Default: ""},
	}
}

func Registry() []CommandSpec {
	env := []EnvVarSpec{
		{Name: "FAKTUROWNIA_PROFILE", Description: "Select a profile unless --profile is provided"},
		{Name: "FAKTUROWNIA_URL", Description: "Override the base account URL from any profile"},
		{Name: "FAKTUROWNIA_API_TOKEN", Description: "Override the API token from any profile"},
	}

	exitCodes := []ExitCodeSpec{
		{Code: 0, Description: "success"},
		{Code: 2, Description: "usage or validation error"},
		{Code: 3, Description: "not found"},
		{Code: 4, Description: "authentication or permission failure"},
		{Code: 5, Description: "conflict"},
		{Code: 6, Description: "network or timeout failure"},
		{Code: 7, Description: "reserved for throttling or retry budget exhaustion"},
		{Code: 8, Description: "remote API rejected the request"},
		{Code: 9, Description: "internal CLI failure"},
	}

	return []CommandSpec{
		{
			Noun:  "auth",
			Verb:  "login",
			Use:   "login --url URL|--prefix PREFIX --api-token TOKEN",
			Short: "Persist a Fakturownia profile and API token",
			Examples: []string{
				"fakturownia auth login --prefix acme --api-token $FAKTUROWNIA_API_TOKEN",
				"fakturownia auth login --url https://acme.fakturownia.pl --api-token TOKEN --profile work --set-default",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			LocalFlags: []FlagSpec{
				{Name: "url", Type: "string", Description: "Explicit HTTPS account URL"},
				{Name: "prefix", Type: "string", Description: "Account prefix such as acme"},
				{Name: "api-token", Type: "string", Description: "Fakturownia API token", Required: true},
				{Name: "set-default", Type: "bool", Description: "Make the saved profile the default selection", Default: "false"},
			},
			DataPrototype: auth.LoginResult{},
		},
		{
			Noun:  "auth",
			Verb:  "exchange",
			Use:   "exchange --login LOGIN --password PASSWORD",
			Short: "Exchange login credentials for account metadata and an API token when available",
			Examples: []string{
				"fakturownia auth exchange --login user@example.com --password secret --json",
				"fakturownia auth exchange --login partner@example.com --password secret --integration-token PARTNER_TOKEN",
				"fakturownia auth exchange --login user@example.com --password secret --save-as work --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "login", Type: "string", Description: "Login or email address", Required: true},
				{Name: "password", Type: "string", Description: "Account password", Required: true},
				{Name: "integration-token", Type: "string", Description: "Integration token for partner API login"},
				{Name: "save-as", Type: "string", Description: "Override the saved profile name; defaults to the returned prefix"},
			},
			DataPrototype: auth.ExchangeResult{},
			Output:        authExchangeOutputSpec(),
		},
		{
			Noun:          "auth",
			Verb:          "status",
			Use:           "status",
			Short:         "Show the resolved authentication state",
			Examples:      []string{"fakturownia auth status", "fakturownia auth status --json"},
			EnvVars:       env,
			OutputModes:   []string{"human", "json"},
			ExitCodes:     exitCodes,
			RawSupported:  false,
			DataPrototype: auth.StatusResult{},
		},
		{
			Noun:  "auth",
			Verb:  "logout",
			Use:   "logout --yes",
			Short: "Remove a persisted profile and token",
			Examples: []string{
				"fakturownia auth logout --profile work --yes",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			LocalFlags: []FlagSpec{
				{Name: "yes", Type: "bool", Description: "Confirm profile removal", Required: true, Default: "false"},
			},
			DataPrototype: auth.LogoutResult{},
		},
		{
			Noun:  "account",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a system account from the full upstream request object",
			Examples: []string{
				`fakturownia account create --input '{"account":{"prefix":"acme"},"user":{"login":"owner","email":"owner@example.com","password":"secret"},"company":{"name":"Acme"}}' --json`,
				`fakturownia account create --input @account-create.json --save-as acme`,
				`fakturownia account create --input @account-create.json --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Full account request JSON as inline JSON, @file, or - for stdin", Required: true},
				{Name: "save-as", Type: "string", Description: "Persist the returned credentials under this profile name"},
			},
			DataPrototype: account.CreateResponse{},
			Output:        accountCreateOutputSpec(),
			RequestBody:   accountCreateRequestBodySpec(),
		},
		{
			Noun:  "account",
			Verb:  "get",
			Use:   "get",
			Short: "Fetch current system-account metadata",
			Examples: []string{
				"fakturownia account get --json",
				"fakturownia account get --integration-token PARTNER_TOKEN",
				"fakturownia account get --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "integration-token", Type: "string", Description: "Integration token used when requesting the current api_token"},
			},
			DataPrototype: account.GetResponse{},
			Output:        accountGetOutputSpec(),
		},
		{
			Noun:  "account",
			Verb:  "delete",
			Use:   "delete --yes",
			Short: "Request deletion of the current system account",
			Examples: []string{
				"fakturownia account delete --yes --json",
				"fakturownia account delete --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "yes", Type: "bool", Description: "Confirm account deletion request", Required: true, Default: "false"},
			},
			DataPrototype: account.DeleteResponse{},
			Output:        accountDeleteOutputSpec(),
		},
		{
			Noun:  "account",
			Verb:  "unlink",
			Use:   "unlink --prefix PREFIX...",
			Short: "Unlink system accounts from a partner integration",
			Examples: []string{
				"fakturownia account unlink --prefix acme --json",
				"fakturownia account unlink --prefix acme --prefix beta --integration-token PARTNER_TOKEN",
				"fakturownia account unlink --prefix acme,beta --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "prefix", Type: "string[]", Description: "Account prefix to unlink; may be repeated or comma-separated", Required: true, Repeatable: true},
				{Name: "integration-token", Type: "string", Description: "Integration token forwarded when required by the upstream integration"},
			},
			DataPrototype: account.UnlinkResponse{},
			Output:        accountUnlinkOutputSpec(),
		},
		{
			Noun:  "department",
			Verb:  "list",
			Use:   "list",
			Short: "List departments",
			Examples: []string{
				"fakturownia department list --json",
				"fakturownia department list --columns id,name,shortcut,tax_no",
				"fakturownia department list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
			},
			DataPrototype: []map[string]any{},
			Output:        departmentListOutputSpec(),
		},
		{
			Noun:  "department",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single department by ID",
			Examples: []string{
				"fakturownia department get --id 10",
				"fakturownia department get --id 10 --fields id,name,shortcut --json",
				"fakturownia department get --id 10 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Department ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        departmentGetOutputSpec("department get"),
		},
		{
			Noun:  "department",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a department",
			Examples: []string{
				`fakturownia department create --input '{"name":"Sales","shortcut":"SALES","tax_no":"123-456-78-90"}' --json`,
				"fakturownia department create --input @department.json",
				`fakturownia department create --input '{"name":"Sales"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Department JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        departmentGetOutputSpec("department create"),
			RequestBody:   departmentRequestBodySpec(),
		},
		{
			Noun:  "department",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a department",
			Examples: []string{
				`fakturownia department update --id 10 --input '{"shortcut":"SALES"}' --json`,
				"fakturownia department update --id 10 --input @department-update.json",
				`fakturownia department update --id 10 --input '{"name":"Sales"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Department ID", Required: true},
				{Name: "input", Type: "string", Description: "Department JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        departmentGetOutputSpec("department update"),
			RequestBody:   departmentRequestBodySpec(),
		},
		{
			Noun:  "department",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete a department",
			Examples: []string{
				"fakturownia department delete --id 10 --yes --json",
				"fakturownia department delete --id 10 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Department ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm department deletion", Required: true, Default: "false"},
			},
			DataPrototype: department.DeleteResponse{},
		},
		{
			Noun:  "department",
			Verb:  "set-logo",
			Use:   "set-logo --id ID --file PATH|-",
			Short: "Upload a department logo",
			Examples: []string{
				"fakturownia department set-logo --id 10 --file ./logo.png --json",
				"cat ./logo.png | fakturownia department set-logo --id 10 --file - --name logo.png --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Department ID", Required: true},
				{Name: "file", Type: "string", Description: "Logo file path or - for stdin", Required: true},
				{Name: "name", Type: "string", Description: "Override the uploaded file name; required when --file - is used"},
			},
			DataPrototype: department.SetLogoResponse{},
			Output:        departmentGetOutputSpec("department set-logo"),
		},
		{
			Noun:  "issuer",
			Verb:  "list",
			Use:   "list",
			Short: "List issuers",
			Examples: []string{
				"fakturownia issuer list --json",
				"fakturownia issuer list --columns id,name,tax_no",
				"fakturownia issuer list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
			},
			DataPrototype: []map[string]any{},
			Output:        issuerListOutputSpec(),
		},
		{
			Noun:  "issuer",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single issuer by ID",
			Examples: []string{
				"fakturownia issuer get --id 3",
				"fakturownia issuer get --id 3 --fields id,name,tax_no --json",
				"fakturownia issuer get --id 3 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Issuer ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        issuerGetOutputSpec("issuer get"),
		},
		{
			Noun:  "issuer",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create an issuer",
			Examples: []string{
				`fakturownia issuer create --input '{"name":"HQ","tax_no":"1234567890"}' --json`,
				"fakturownia issuer create --input @issuer.json",
				`fakturownia issuer create --input '{"name":"HQ"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Issuer JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        issuerGetOutputSpec("issuer create"),
			RequestBody:   issuerRequestBodySpec(),
		},
		{
			Noun:  "issuer",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update an issuer",
			Examples: []string{
				`fakturownia issuer update --id 3 --input '{"tax_no":"1234567890"}' --json`,
				"fakturownia issuer update --id 3 --input @issuer-update.json",
				`fakturownia issuer update --id 3 --input '{"name":"HQ"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Issuer ID", Required: true},
				{Name: "input", Type: "string", Description: "Issuer JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        issuerGetOutputSpec("issuer update"),
			RequestBody:   issuerRequestBodySpec(),
		},
		{
			Noun:  "issuer",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete an issuer",
			Examples: []string{
				"fakturownia issuer delete --id 3 --yes --json",
				"fakturownia issuer delete --id 3 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Issuer ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm issuer deletion", Required: true, Default: "false"},
			},
			DataPrototype: issuer.DeleteResponse{},
		},
		{
			Noun:  "user",
			Verb:  "create",
			Use:   "create --input -|@file|JSON --integration-token TOKEN",
			Short: "Create or invite an account user",
			Examples: []string{
				`fakturownia user create --integration-token PARTNER_TOKEN --input '{"invite":true,"email":"user@example.com","role":"member"}' --json`,
				`fakturownia user create --integration-token PARTNER_TOKEN --input '{"invite":false,"email":"user@example.com","password":"secret","role":"admin","department_ids":[1,2]}' --json`,
				`fakturownia user create --integration-token PARTNER_TOKEN --input @user.json --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "User JSON input as inline JSON, @file, or - for stdin", Required: true},
				{Name: "integration-token", Type: "string", Description: "Integration token required by the upstream add_user endpoint", Required: true},
			},
			DataPrototype: map[string]any{},
			RequestBody:   userCreateRequestBodySpec(),
		},
		{
			Noun:  "webhook",
			Verb:  "list",
			Use:   "list",
			Short: "List webhooks",
			Examples: []string{
				"fakturownia webhook list --json",
				"fakturownia webhook list --columns id,kind,url,active",
				"fakturownia webhook list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
			},
			DataPrototype: []map[string]any{},
			Output:        webhookListOutputSpec(),
		},
		{
			Noun:  "webhook",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single webhook by ID",
			Examples: []string{
				"fakturownia webhook get --id 7",
				"fakturownia webhook get --id 7 --fields id,kind,url,active --json",
				"fakturownia webhook get --id 7 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Webhook ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        webhookGetOutputSpec("webhook get"),
		},
		{
			Noun:  "webhook",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a webhook from the full top-level request object",
			Examples: []string{
				`fakturownia webhook create --input '{"kind":"invoice:create","url":"https://example.com/hook","api_token":"secret","active":true}' --json`,
				"fakturownia webhook create --input @webhook.json",
				`fakturownia webhook create --input '{"kind":"invoice:create","url":"https://example.com/hook"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Full webhook request JSON as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        webhookGetOutputSpec("webhook create"),
			RequestBody:   webhookRequestBodySpec(),
		},
		{
			Noun:  "webhook",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a webhook from the full top-level request object",
			Examples: []string{
				`fakturownia webhook update --id 7 --input '{"active":false}' --json`,
				"fakturownia webhook update --id 7 --input @webhook-update.json",
				`fakturownia webhook update --id 7 --input '{"url":"https://example.com/hook"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Webhook ID", Required: true},
				{Name: "input", Type: "string", Description: "Full webhook request JSON as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        webhookGetOutputSpec("webhook update"),
			RequestBody:   webhookRequestBodySpec(),
		},
		{
			Noun:  "webhook",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete a webhook",
			Examples: []string{
				"fakturownia webhook delete --id 7 --yes --json",
				"fakturownia webhook delete --id 7 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Webhook ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm webhook deletion", Required: true, Default: "false"},
			},
			DataPrototype: webhook.DeleteResponse{},
		},
		{
			Noun:  "category",
			Verb:  "list",
			Use:   "list",
			Short: "List categories",
			Examples: []string{
				"fakturownia category list --json",
				"fakturownia category list --columns id,name,description",
				"fakturownia category list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
			},
			DataPrototype: []map[string]any{},
			Output:        categoryListOutputSpec(),
		},
		{
			Noun:  "category",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single category by ID",
			Examples: []string{
				"fakturownia category get --id 100",
				"fakturownia category get --id 100 --fields id,name,description --json",
				"fakturownia category get --id 100 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Category ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        categoryGetOutputSpec("category get"),
		},
		{
			Noun:  "category",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a category",
			Examples: []string{
				`fakturownia category create --input '{"name":"my_category","description":null}' --json`,
				"fakturownia category create --input @category.json",
				`printf '%s\n' '{"name":"my_category"}' | fakturownia category create --input - --json`,
				`fakturownia category create --input '{"name":"my_category"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Category JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        categoryGetOutputSpec("category create"),
			RequestBody:   categoryRequestBodySpec(),
		},
		{
			Noun:  "category",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a category",
			Examples: []string{
				`fakturownia category update --id 100 --input '{"description":"new_description"}' --json`,
				"fakturownia category update --id 100 --input @category-update.json",
				`fakturownia category update --id 100 --input '{"name":"my_category"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Category ID", Required: true},
				{Name: "input", Type: "string", Description: "Category JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        categoryGetOutputSpec("category update"),
			RequestBody:   categoryRequestBodySpec(),
		},
		{
			Noun:  "category",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete a category",
			Examples: []string{
				"fakturownia category delete --id 100 --yes --json",
				"fakturownia category delete --id 100 --yes",
				"fakturownia category delete --id 100 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Category ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm category deletion", Required: true, Default: "false"},
			},
			DataPrototype: category.DeleteResponse{},
		},
		{
			Noun:  "invoice",
			Verb:  "list",
			Use:   "list",
			Short: "List invoices",
			Examples: []string{
				"fakturownia invoice list --json",
				"fakturownia invoice list --period this_month --columns id,number,buyer_name,price_gross",
				"fakturownia invoice list --include-positions --fields number,positions[].name --json",
				"fakturownia invoice list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
				{Name: "period", Type: "string", Description: "Date period filter", Enum: []string{"last_12_months", "this_month", "last_30_days", "last_month", "this_year", "last_year", "all", "more"}},
				{Name: "date-from", Type: "string", Description: "Lower date bound for period=more"},
				{Name: "date-to", Type: "string", Description: "Upper date bound for period=more"},
				{Name: "include-positions", Type: "bool", Description: "Include invoice positions", Default: "false"},
				{Name: "client-id", Type: "string", Description: "Filter by client ID"},
				{Name: "invoice-ids", Type: "string[]", Description: "Filter by specific invoice IDs", Repeatable: true},
				{Name: "number", Type: "string", Description: "Filter by invoice number"},
				{Name: "kind", Type: "string[]", Description: "Filter by invoice kind; one value uses kind= and repeated values use kinds[]", Repeatable: true},
				{Name: "search-date-type", Type: "string", Description: "Date field to search by", Enum: []string{"issue_date", "paid_date", "transaction_date"}},
				{Name: "order", Type: "string", Description: "Sort order"},
				{Name: "income", Type: "string", Description: "Income selector", Enum: []string{"yes", "no"}},
			},
			DataPrototype: []map[string]any{},
			Output:        invoiceListOutputSpec(),
		},
		{
			Noun:  "invoice",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single invoice by ID",
			Examples: []string{
				"fakturownia invoice get --id 123",
				"fakturownia invoice get --id 123 --additional-field cancel_reason --json",
				"fakturownia invoice get --id 123 --include descriptions --fields descriptions[].content --json",
				"fakturownia invoice get --id 123 --correction-positions full --json",
				"fakturownia invoice get --id 123 --fields id,number,status --json",
				"fakturownia invoice get --id 123 --fields id,number,bank_accounts[].bank_account_number --json",
				"fakturownia invoice get --id 123 --fields number,positions[].name --json",
				"fakturownia invoice get --id 123 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "include", Type: "string[]", Description: "Request upstream invoice includes such as descriptions", Repeatable: true},
				{Name: "additional-field", Type: "string[]", Description: "Request additional upstream invoice fields such as cancel_reason, corrected_content_before, corrected_content_after, or connected_payments", Repeatable: true},
				{Name: "correction-positions", Type: "string", Description: "Request correction position details such as full"},
			},
			DataPrototype: map[string]any{},
			Output:        invoiceGetOutputSpec("invoice get"),
		},
		{
			Noun:  "invoice",
			Verb:  "download",
			Use:   "download --id ID",
			Short: "Download a single invoice PDF",
			Examples: []string{
				"fakturownia invoice download --id 123 --dir ./invoices",
				"fakturownia invoice download --id 123 --path ./invoice-123.pdf --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "path", Type: "string", Description: "Explicit output file path"},
				{Name: "dir", Type: "string", Description: "Output directory for the downloaded file"},
				{Name: "print-option", Type: "string", Description: "PDF print option", Enum: []string{"original", "copy", "original_and_copy", "duplicate"}},
			},
			DataPrototype: invoice.DownloadResponse{},
			Output: &OutputSpec{
				Shape:     "file",
				OpenEnded: false,
				Notes: []string{
					"download writes PDF bytes to disk and returns CLI-generated metadata",
				},
			},
		},
		{
			Noun:  "invoice",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create an invoice",
			Examples: []string{
				`fakturownia invoice create --input '{"kind":"vat","client_id":1,"positions":[{"product_id":1,"quantity":2}]}' --json`,
				`fakturownia invoice create --input '{"kind":"vat","buyer_name":"Klient ABC","bank_account_id":100,"buyer_mass_payment_code":"ABC-123","positions":[{"name":"Usługa","quantity":1,"total_price_gross":1230,"tax":23}]}' --json`,
				`fakturownia invoice create --input '{"copy_invoice_from":42,"kind":"vat"}' --json`,
				`fakturownia invoice create --gov-save-and-send --input '{"kind":"vat","buyer_company":true,"seller_tax_no":"5252445767","seller_street":"ul. Przykładowa 10","seller_post_code":"00-001","seller_city":"Warszawa","buyer_name":"Klient ABC Sp. z o.o.","buyer_tax_no":"9876543210","positions":[{"name":"Usługa","quantity":1,"total_price_gross":1230,"tax":23}]}' --json`,
				`fakturownia invoice create --input '{"kind":"vat","seller_country":"PL","buyer_country":"FR","use_oss":true,"positions":[{"name":"Produkt","tax":20,"total_price_gross":50,"quantity":1}]}' --identify-oss --json`,
				`fakturownia invoice create --input '{"kind":"vat","positions":[{"name":"towar","quantity":1,"total_price_gross":123}]}' --fill-default-descriptions --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Invoice JSON input as inline JSON, @file, or - for stdin", Required: true},
				{Name: "identify-oss", Type: "bool", Description: "Validate OSS eligibility before marking the invoice as OSS", Default: "false"},
				{Name: "fill-default-descriptions", Type: "bool", Description: "Include default account descriptions on the created invoice", Default: "false"},
				{Name: "correction-positions", Type: "string", Description: "Pass a correction positions companion option such as full"},
				{Name: "gov-save-and-send", Type: "bool", Description: "Save the invoice and immediately queue it for KSeF submission", Default: "false"},
			},
			DataPrototype: map[string]any{},
			Output:        invoiceGetOutputSpec("invoice create"),
			RequestBody:   invoiceCreateRequestBodySpec(),
		},
		{
			Noun:  "invoice",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update an invoice",
			Examples: []string{
				`fakturownia invoice update --id 111 --input '{"buyer_name":"Nowa nazwa klienta Sp. z o.o."}' --json`,
				`fakturownia invoice update --id 111 --input '{"positions":[{"id":32649087,"name":"test"}]}' --json`,
				`fakturownia invoice update --id 111 --input '{"positions":[{"id":32649087,"_destroy":1}]}' --json`,
				`fakturownia invoice update --id 111 --gov-save-and-send --input '{"buyer_company":true,"buyer_tax_no_kind":"nip_ue","buyer_tax_no":"DE123456789"}' --json`,
				`printf '%s\n' '{"show_attachments":true}' | fakturownia invoice update --id 111 --input - --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "input", Type: "string", Description: "Invoice JSON input as inline JSON, @file, or - for stdin", Required: true},
				{Name: "identify-oss", Type: "bool", Description: "Validate OSS eligibility before marking the invoice as OSS", Default: "false"},
				{Name: "fill-default-descriptions", Type: "bool", Description: "Include default account descriptions on the updated invoice", Default: "false"},
				{Name: "correction-positions", Type: "string", Description: "Pass a correction positions companion option such as full"},
				{Name: "gov-save-and-send", Type: "bool", Description: "Save the invoice and immediately queue it for KSeF submission", Default: "false"},
			},
			DataPrototype: map[string]any{},
			Output:        invoiceGetOutputSpec("invoice update"),
			RequestBody:   invoiceUpdateRequestBodySpec(),
		},
		{
			Noun:  "invoice",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete an invoice",
			Examples: []string{
				"fakturownia invoice delete --id 111 --yes --json",
				"fakturownia invoice delete --id 111 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm invoice deletion", Required: true, Default: "false"},
			},
			DataPrototype: invoice.DeleteResponse{},
		},
		{
			Noun:  "invoice",
			Verb:  "send-email",
			Use:   "send-email --id ID",
			Short: "Send an invoice by email",
			Examples: []string{
				"fakturownia invoice send-email --id 100 --json",
				"fakturownia invoice send-email --id 100 --email-to billing@example.com --email-pdf --json",
				"fakturownia invoice send-email --id 100 --email-to billing@example.com --update-buyer-email --print-option original --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "email-to", Type: "string[]", Description: "Override recipients for the invoice email", Repeatable: true},
				{Name: "email-cc", Type: "string[]", Description: "Override CC recipients for the invoice email", Repeatable: true},
				{Name: "email-pdf", Type: "bool", Description: "Attach the invoice PDF", Default: "false"},
				{Name: "update-buyer-email", Type: "bool", Description: "Update the invoice buyer or recipient email when email-to is provided", Default: "false"},
				{Name: "print-option", Type: "string", Description: "PDF print option", Enum: []string{"original", "copy", "original_and_copy", "duplicate"}},
			},
			DataPrototype: invoice.SendEmailResponse{},
		},
		{
			Noun:  "invoice",
			Verb:  "send-gov",
			Use:   "send-gov --id ID",
			Short: "Queue an existing invoice for KSeF submission",
			Examples: []string{
				"fakturownia invoice send-gov --id 100 --json",
				"fakturownia invoice send-gov --id 100 --raw",
				"fakturownia invoice send-gov --id 100 --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        invoiceGetOutputSpec("invoice send-gov"),
		},
		{
			Noun:  "invoice",
			Verb:  "change-status",
			Use:   "change-status --id ID --status STATUS",
			Short: "Change an invoice status",
			Examples: []string{
				"fakturownia invoice change-status --id 111 --status sent --json",
				"fakturownia invoice change-status --id 111 --status paid --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "status", Type: "string", Description: "Target invoice status", Required: true},
			},
			DataPrototype: invoice.ChangeStatusResponse{},
		},
		{
			Noun:  "invoice",
			Verb:  "cancel",
			Use:   "cancel --id ID --yes",
			Short: "Cancel an invoice",
			Examples: []string{
				"fakturownia invoice cancel --id 111 --yes --json",
				`fakturownia invoice cancel --id 111 --yes --reason 'Powód anulowania' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "reason", Type: "string", Description: "Optional cancellation reason"},
				{Name: "yes", Type: "bool", Description: "Confirm invoice cancellation", Required: true, Default: "false"},
			},
			DataPrototype: invoice.CancelResponse{},
		},
		{
			Noun:  "invoice",
			Verb:  "public-link",
			Use:   "public-link --id ID",
			Short: "Derive public invoice and PDF links from the invoice token",
			Examples: []string{
				"fakturownia invoice public-link --id 100 --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
			},
			DataPrototype: invoice.PublicLinkResponse{},
		},
		{
			Noun:  "invoice",
			Verb:  "add-attachment",
			Use:   "add-attachment --id ID --file PATH|-",
			Short: "Upload and attach a file to an invoice",
			Examples: []string{
				"fakturownia invoice add-attachment --id 111 --file ./scan.pdf --json",
				"cat ./scan.pdf | fakturownia invoice add-attachment --id 111 --file - --name scan.pdf --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "file", Type: "string", Description: "Attachment file path or - for stdin", Required: true},
				{Name: "name", Type: "string", Description: "Attachment file name; required when --file - is used"},
			},
			DataPrototype: invoice.AddAttachmentResponse{},
			Output: &OutputSpec{
				Shape:     "object",
				OpenEnded: false,
				Notes: []string{
					"add-attachment uses a multi-step helper flow: fetch credentials, upload to the returned third-party URL, then attach the uploaded file to the invoice",
				},
			},
		},
		{
			Noun:  "invoice",
			Verb:  "download-attachments",
			Use:   "download-attachments --id ID",
			Short: "Download all invoice attachments as a ZIP archive",
			Examples: []string{
				"fakturownia invoice download-attachments --id 111 --dir ./attachments",
				"fakturownia invoice download-attachments --id 111 --path ./invoice-111-attachments.zip --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "path", Type: "string", Description: "Explicit output file path"},
				{Name: "dir", Type: "string", Description: "Output directory for the downloaded ZIP file"},
			},
			DataPrototype: invoice.DownloadAttachmentsResponse{},
			Output: &OutputSpec{
				Shape:     "file",
				OpenEnded: false,
				Notes: []string{
					"download-attachments writes ZIP bytes to disk and returns CLI-generated metadata",
				},
			},
		},
		{
			Noun:  "invoice",
			Verb:  "download-attachment",
			Use:   "download-attachment --id ID --kind KIND",
			Short: "Download a single invoice attachment by kind",
			Examples: []string{
				"fakturownia invoice download-attachment --id 111 --kind gov",
				"fakturownia invoice download-attachment --id 111 --kind gov_upo --dir ./attachments --json",
				"fakturownia invoice download-attachment --id 111 --kind custom --path ./attachment.bin --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
				{Name: "kind", Type: "string", Description: "Attachment kind such as gov or gov_upo", Required: true},
				{Name: "path", Type: "string", Description: "Explicit output file path"},
				{Name: "dir", Type: "string", Description: "Output directory for the downloaded attachment"},
			},
			DataPrototype: invoice.DownloadAttachmentResponse{},
			Output: &OutputSpec{
				Shape:     "file",
				OpenEnded: false,
				Notes: []string{
					"download-attachment writes the requested attachment bytes to disk and returns CLI-generated metadata",
					"`kind=gov` maps to the KSeF invoice XML and `kind=gov_upo` maps to the KSeF UPO XML",
				},
			},
		},
		{
			Noun:  "invoice",
			Verb:  "fiscal-print",
			Use:   "fiscal-print --invoice-id ID...",
			Short: "Trigger a fiscal printer job for one or more invoices",
			Examples: []string{
				"fakturownia invoice fiscal-print --invoice-id 111 --invoice-id 112 --json",
				"fakturownia invoice fiscal-print --invoice-id 111 --printer DRUKARKA --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "invoice-id", Type: "string[]", Description: "Invoice ID to send to fiscal print", Required: true, Repeatable: true},
				{Name: "printer", Type: "string", Description: "Fiscal printer name"},
			},
			DataPrototype: invoice.FiscalPrintResponse{},
		},
		{
			Noun:  "recurring",
			Verb:  "list",
			Use:   "list",
			Short: "List recurring invoice definitions",
			Examples: []string{
				"fakturownia recurring list --json",
				"fakturownia recurring list --columns id,name,every,next_invoice_date,send_email",
				"fakturownia recurring list --raw",
			},
			EnvVars:       env,
			OutputModes:   []string{"human", "json"},
			ExitCodes:     exitCodes,
			RawSupported:  true,
			DataPrototype: []map[string]any{},
			Output:        recurringListOutputSpec(),
		},
		{
			Noun:  "recurring",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a recurring invoice definition",
			Examples: []string{
				`fakturownia recurring create --input '{"name":"Nazwa cyklicznosci","invoice_id":1,"start_date":"2016-01-01","every":"1m","send_email":true}' --json`,
				`fakturownia recurring create --input '{"name":"Nazwa cyklicznosci","invoice_id":1,"every":"1m"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Recurring JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        recurringGetOutputSpec("recurring create"),
			RequestBody:   recurringRequestBodySpec(),
		},
		{
			Noun:  "recurring",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a recurring invoice definition",
			Examples: []string{
				`fakturownia recurring update --id 111 --input '{"next_invoice_date":"2016-02-01"}' --json`,
				`fakturownia recurring update --id 111 --input '{"next_invoice_date":"2016-02-01"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Recurring definition ID", Required: true},
				{Name: "input", Type: "string", Description: "Recurring JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        recurringGetOutputSpec("recurring update"),
			RequestBody:   recurringRequestBodySpec(),
		},
		{
			Noun:  "client",
			Verb:  "list",
			Use:   "list",
			Short: "List clients",
			Examples: []string{
				"fakturownia client list --json",
				"fakturownia client list --name Acme --columns id,name,email,country",
				"fakturownia client list --external-id ext-123 --json",
				"fakturownia client list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
				{Name: "name", Type: "string", Description: "Filter by client name"},
				{Name: "email", Type: "string", Description: "Filter by client email"},
				{Name: "shortcut", Type: "string", Description: "Filter by client shortcut"},
				{Name: "tax-no", Type: "string", Description: "Filter by client tax number"},
				{Name: "external-id", Type: "string", Description: "Filter by external client ID"},
			},
			DataPrototype: []map[string]any{},
			Output:        clientListOutputSpec(),
		},
		{
			Noun:  "client",
			Verb:  "get",
			Use:   "get --id ID | --external-id X",
			Short: "Fetch a single client by ID or external ID",
			Examples: []string{
				"fakturownia client get --id 123",
				"fakturownia client get --external-id ext-123 --json",
				"fakturownia client get --id 123 --fields id,name,email --json",
				"fakturownia client get --id 123 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Client ID"},
				{Name: "external-id", Type: "string", Description: "External client ID"},
			},
			DataPrototype: map[string]any{},
			Output:        clientGetOutputSpec("client get"),
		},
		{
			Noun:  "client",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a client",
			Examples: []string{
				`fakturownia client create --input '{"name":"Acme","email":"billing@example.com"}' --json`,
				"fakturownia client create --input @client.json",
				`printf '%s\n' '{"name":"Acme","company":"1"}' | fakturownia client create --input - --json`,
				`fakturownia client create --input '{"name":"Acme"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Client JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        clientGetOutputSpec("client create"),
			RequestBody:   clientRequestBodySpec(),
		},
		{
			Noun:  "client",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a client",
			Examples: []string{
				`fakturownia client update --id 123 --input '{"email":"billing@example.com"}' --json`,
				"fakturownia client update --id 123 --input @client-update.json",
				`printf '%s\n' '{"phone":"123456789"}' | fakturownia client update --id 123 --input - --json`,
				`fakturownia client update --id 123 --input '{"city":"Warsaw"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Client ID", Required: true},
				{Name: "input", Type: "string", Description: "Client JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        clientGetOutputSpec("client update"),
			RequestBody:   clientRequestBodySpec(),
		},
		{
			Noun:  "client",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete a client",
			Examples: []string{
				"fakturownia client delete --id 123 --yes --json",
				"fakturownia client delete --id 123 --yes",
				"fakturownia client delete --id 123 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Client ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm client deletion", Required: true, Default: "false"},
			},
			DataPrototype: client.DeleteResponse{},
		},
		{
			Noun:  "payment",
			Verb:  "list",
			Use:   "list",
			Short: "List payments",
			Examples: []string{
				"fakturownia payment list --json",
				"fakturownia payment list --include invoices --json",
				"fakturownia payment list --columns id,name,price,paid,kind",
				"fakturownia payment list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
				{Name: "include", Type: "string[]", Description: "README-backed include such as invoices", Repeatable: true, Enum: []string{"invoices"}},
			},
			DataPrototype: []map[string]any{},
			Output:        paymentListOutputSpec(),
		},
		{
			Noun:  "payment",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single payment by ID",
			Examples: []string{
				"fakturownia payment get --id 555",
				"fakturownia payment get --id 555 --fields id,name,price,paid --json",
				"fakturownia payment get --id 555 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Payment ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        paymentGetOutputSpec("payment get"),
		},
		{
			Noun:  "payment",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a payment",
			Examples: []string{
				`fakturownia payment create --input '{"name":"Payment 001","price":100.05,"invoice_id":null,"paid":true,"kind":"api"}' --json`,
				`fakturownia payment create --input '{"name":"Payment 003","price":200,"invoice_ids":[555,666],"paid":true,"kind":"api"}' --json`,
				"fakturownia payment create --input @payment.json",
				`fakturownia payment create --input '{"name":"Payment 001"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Payment JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        paymentGetOutputSpec("payment create"),
			RequestBody:   paymentRequestBodySpec(),
		},
		{
			Noun:  "payment",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a payment",
			Examples: []string{
				`fakturownia payment update --id 555 --input '{"name":"New payment name","price":100}' --json`,
				"fakturownia payment update --id 555 --input @payment-update.json",
				`fakturownia payment update --id 555 --input '{"name":"New payment name"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Payment ID", Required: true},
				{Name: "input", Type: "string", Description: "Payment JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        paymentGetOutputSpec("payment update"),
			RequestBody:   paymentRequestBodySpec(),
		},
		{
			Noun:  "payment",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete a payment",
			Examples: []string{
				"fakturownia payment delete --id 555 --yes --json",
				"fakturownia payment delete --id 555 --yes",
				"fakturownia payment delete --id 555 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Payment ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm payment deletion", Required: true, Default: "false"},
			},
			DataPrototype: payment.DeleteResponse{},
		},
		{
			Noun:  "bank-account",
			Verb:  "list",
			Use:   "list",
			Short: "List bank accounts",
			Examples: []string{
				"fakturownia bank-account list --json",
				"fakturownia bank-account list --columns id,name,bank_account_number,bank_currency,default",
				"fakturownia bank-account list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
			},
			DataPrototype: []map[string]any{},
			Output:        bankAccountListOutputSpec(),
		},
		{
			Noun:  "bank-account",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single bank account by ID",
			Examples: []string{
				"fakturownia bank-account get --id 100",
				"fakturownia bank-account get --id 100 --fields id,name,bank_account_number --json",
				"fakturownia bank-account get --id 100 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Bank account ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        bankAccountGetOutputSpec("bank-account get"),
		},
		{
			Noun:  "bank-account",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a bank account",
			Examples: []string{
				`fakturownia bank-account create --input '{"name":"Rachunek główny PLN","bank_account_number":"PL61 1090 1014 0000 0712 1981 2874","bank_name":"Santander Bank Polska","bank_currency":"PLN","default":true}' --json`,
				`fakturownia bank-account create --input '{"name":"Rachunek działu","bank_account_version_departments":[{"department_id":5,"main_on_department":true,"show_on_invoice":true}]}' --json`,
				"fakturownia bank-account create --input @bank-account.json",
				`fakturownia bank-account create --input '{"name":"Rachunek testowy"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Bank account JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        bankAccountGetOutputSpec("bank-account create"),
			RequestBody:   bankAccountRequestBodySpec(),
		},
		{
			Noun:  "bank-account",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a bank account",
			Examples: []string{
				`fakturownia bank-account update --id 100 --input '{"default":false}' --json`,
				`fakturownia bank-account update --id 100 --input '{"bank_account_version_departments":[{"department_id":5,"remove":true}]}' --json`,
				"fakturownia bank-account update --id 100 --input @bank-account-update.json",
				`fakturownia bank-account update --id 100 --input '{"bank_swift":"ABCDPLPW"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Bank account ID", Required: true},
				{Name: "input", Type: "string", Description: "Bank account JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        bankAccountGetOutputSpec("bank-account update"),
			RequestBody:   bankAccountRequestBodySpec(),
		},
		{
			Noun:  "bank-account",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete a bank account",
			Examples: []string{
				"fakturownia bank-account delete --id 100 --yes --json",
				"fakturownia bank-account delete --id 100 --yes",
				"fakturownia bank-account delete --id 100 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Bank account ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm bank account deletion", Required: true, Default: "false"},
			},
			DataPrototype: bankaccount.DeleteResponse{},
		},
		{
			Noun:  "product",
			Verb:  "list",
			Use:   "list",
			Short: "List products",
			Examples: []string{
				"fakturownia product list --json",
				"fakturownia product list --date-from 2025-11-01 --json",
				"fakturownia product list --warehouse-id 7 --columns id,name,code,stock_level",
				"fakturownia product list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
				{Name: "date-from", Type: "string", Description: "Filter products added or changed since a date such as 2025-11-01"},
				{Name: "warehouse-id", Type: "string", Description: "Show stock levels for a specific warehouse"},
			},
			DataPrototype: []map[string]any{},
			Output:        productListOutputSpec(),
		},
		{
			Noun:  "product",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single product by ID",
			Examples: []string{
				"fakturownia product get --id 100",
				"fakturownia product get --id 100 --warehouse-id 7 --json",
				"fakturownia product get --id 100 --fields id,name,price_gross,stock_level --json",
				"fakturownia product get --id 100 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Product ID", Required: true},
				{Name: "warehouse-id", Type: "string", Description: "Show stock level for a specific warehouse"},
			},
			DataPrototype: map[string]any{},
			Output:        productGetOutputSpec("product get"),
		},
		{
			Noun:  "product",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a product",
			Examples: []string{
				`fakturownia product create --input '{"name":"Widget","code":"W001","price_net":"100","tax":"23"}' --json`,
				"fakturownia product create --input @product.json",
				`printf '%s\n' '{"name":"Bundle","package":"1","package_products_details":{"0":{"id":5,"quantity":1}}}' | fakturownia product create --input - --json`,
				`fakturownia product create --input '{"name":"Widget"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Product JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        productGetOutputSpec("product create"),
			RequestBody:   productCreateRequestBodySpec(),
		},
		{
			Noun:  "product",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a product",
			Examples: []string{
				`fakturownia product update --id 333 --input '{"price_gross":"102","tax":"23"}' --json`,
				"fakturownia product update --id 333 --input @product-update.json",
				`printf '%s\n' '{"name":"Widget 2"}' | fakturownia product update --id 333 --input - --json`,
				`fakturownia product update --id 333 --input '{"price_gross":"102"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Product ID", Required: true},
				{Name: "input", Type: "string", Description: "Product JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        productGetOutputSpec("product update"),
			RequestBody:   productUpdateRequestBodySpec(),
		},
		{
			Noun:  "price-list",
			Verb:  "list",
			Use:   "list",
			Short: "List price lists",
			Examples: []string{
				"fakturownia price-list list --json",
				"fakturownia price-list list --columns id,name,currency,description",
				"fakturownia price-list list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
			},
			DataPrototype: []map[string]any{},
			Output:        priceListListOutputSpec(),
		},
		{
			Noun:  "price-list",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single price list by ID",
			Examples: []string{
				"fakturownia price-list get --id 8523",
				"fakturownia price-list get --id 8523 --fields id,name,price_list_positions[].price_gross --json",
				"fakturownia price-list get --id 8523 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Price list ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        priceListGetOutputSpec("price-list get"),
		},
		{
			Noun:  "price-list",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a price list",
			Examples: []string{
				`fakturownia price-list create --input '{"name":"Dropshipper","currency":"PLN"}' --json`,
				`fakturownia price-list create --input '{"name":"Dropshipper","price_list_positions_attributes":{"0":{"priceable_id":97149307,"price_gross":"33.16","tax":"23"}}}' --json`,
				"fakturownia price-list create --input @price-list.json",
				`fakturownia price-list create --input '{"name":"Dropshipper"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Price list JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        priceListGetOutputSpec("price-list create"),
			RequestBody:   priceListRequestBodySpec(),
		},
		{
			Noun:  "price-list",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a price list",
			Examples: []string{
				`fakturownia price-list update --id 8523 --input '{"description":"updated"}' --json`,
				"fakturownia price-list update --id 8523 --input @price-list-update.json",
				`fakturownia price-list update --id 8523 --input '{"currency":"EUR"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Price list ID", Required: true},
				{Name: "input", Type: "string", Description: "Price list JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        priceListGetOutputSpec("price-list update"),
			RequestBody:   priceListRequestBodySpec(),
		},
		{
			Noun:  "price-list",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete a price list",
			Examples: []string{
				"fakturownia price-list delete --id 8523 --yes --json",
				"fakturownia price-list delete --id 8523 --yes",
				"fakturownia price-list delete --id 8523 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Price list ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm price list deletion", Required: true, Default: "false"},
			},
			DataPrototype: pricelist.DeleteResponse{},
		},
		{
			Noun:  "warehouse-document",
			Verb:  "list",
			Use:   "list",
			Short: "List warehouse documents",
			Examples: []string{
				"fakturownia warehouse-document list --json",
				"fakturownia warehouse-document list --columns id,kind,number,client_name",
				"fakturownia warehouse-document list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
			},
			DataPrototype: []map[string]any{},
			Output:        warehouseDocumentListOutputSpec(),
		},
		{
			Noun:  "warehouse-document",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single warehouse document by ID",
			Examples: []string{
				"fakturownia warehouse-document get --id 15",
				"fakturownia warehouse-document get --id 15 --fields id,kind,warehouse_actions[].quantity --json",
				"fakturownia warehouse-document get --id 15 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Warehouse document ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        warehouseDocumentGetOutputSpec("warehouse-document get"),
		},
		{
			Noun:  "warehouse-document",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a warehouse document",
			Examples: []string{
				`fakturownia warehouse-document create --input '{"kind":"mm","warehouse_id":1,"warehouse_actions":[{"product_id":7,"quantity":2,"warehouse2_id":3}]}' --json`,
				`fakturownia warehouse-document create --input '{"kind":"wz","client_id":12,"warehouse_actions":[{"product_id":7,"tax":"23","price_net":"100","quantity":2}]}' --json`,
				"fakturownia warehouse-document create --input @warehouse-document.json",
				`fakturownia warehouse-document create --input '{"kind":"mm"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Warehouse document JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        warehouseDocumentGetOutputSpec("warehouse-document create"),
			RequestBody:   warehouseDocumentRequestBodySpec(),
		},
		{
			Noun:  "warehouse-document",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a warehouse document",
			Examples: []string{
				`fakturownia warehouse-document update --id 15 --input '{"invoice_ids":[100,111]}' --json`,
				"fakturownia warehouse-document update --id 15 --input @warehouse-document-update.json",
				`fakturownia warehouse-document update --id 15 --input '{"kind":"pz"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Warehouse document ID", Required: true},
				{Name: "input", Type: "string", Description: "Warehouse document JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        warehouseDocumentGetOutputSpec("warehouse-document update"),
			RequestBody:   warehouseDocumentRequestBodySpec(),
		},
		{
			Noun:  "warehouse-document",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete a warehouse document",
			Examples: []string{
				"fakturownia warehouse-document delete --id 15 --yes --json",
				"fakturownia warehouse-document delete --id 15 --yes",
				"fakturownia warehouse-document delete --id 15 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Warehouse document ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm warehouse document deletion", Required: true, Default: "false"},
			},
			DataPrototype: warehousedocument.DeleteResponse{},
		},
		{
			Noun:  "warehouse",
			Verb:  "list",
			Use:   "list",
			Short: "List warehouses",
			Examples: []string{
				"fakturownia warehouse list --json",
				"fakturownia warehouse list --columns id,name,kind,description",
				"fakturownia warehouse list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
			},
			DataPrototype: []map[string]any{},
			Output:        warehouseListOutputSpec(),
		},
		{
			Noun:  "warehouse",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single warehouse by ID",
			Examples: []string{
				"fakturownia warehouse get --id 1",
				"fakturownia warehouse get --id 1 --fields id,name,description --json",
				"fakturownia warehouse get --id 1 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Warehouse ID", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        warehouseGetOutputSpec("warehouse get"),
		},
		{
			Noun:  "warehouse",
			Verb:  "create",
			Use:   "create --input -|@file|JSON",
			Short: "Create a warehouse",
			Examples: []string{
				`fakturownia warehouse create --input '{"name":"my_warehouse","kind":null,"description":null}' --json`,
				"fakturownia warehouse create --input @warehouse.json",
				`printf '%s\n' '{"name":"my_warehouse"}' | fakturownia warehouse create --input - --json`,
				`fakturownia warehouse create --input '{"name":"my_warehouse"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "input", Type: "string", Description: "Warehouse JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        warehouseGetOutputSpec("warehouse create"),
			RequestBody:   warehouseRequestBodySpec(),
		},
		{
			Noun:  "warehouse",
			Verb:  "update",
			Use:   "update --id ID --input -|@file|JSON",
			Short: "Update a warehouse",
			Examples: []string{
				`fakturownia warehouse update --id 1 --input '{"description":"new_description"}' --json`,
				"fakturownia warehouse update --id 1 --input @warehouse-update.json",
				`printf '%s\n' '{"name":"my_warehouse"}' | fakturownia warehouse update --id 1 --input - --json`,
				`fakturownia warehouse update --id 1 --input '{"description":"new_description"}' --dry-run --json`,
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Warehouse ID", Required: true},
				{Name: "input", Type: "string", Description: "Warehouse JSON input as inline JSON, @file, or - for stdin", Required: true},
			},
			DataPrototype: map[string]any{},
			Output:        warehouseGetOutputSpec("warehouse update"),
			RequestBody:   warehouseRequestBodySpec(),
		},
		{
			Noun:  "warehouse",
			Verb:  "delete",
			Use:   "delete --id ID --yes",
			Short: "Delete a warehouse",
			Examples: []string{
				"fakturownia warehouse delete --id 1 --yes --json",
				"fakturownia warehouse delete --id 1 --yes",
				"fakturownia warehouse delete --id 1 --yes --dry-run --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Warehouse ID", Required: true},
				{Name: "yes", Type: "bool", Description: "Confirm warehouse deletion", Required: true, Default: "false"},
			},
			DataPrototype: warehouse.DeleteResponse{},
		},
		{
			Noun:  "warehouse-action",
			Verb:  "list",
			Use:   "list",
			Short: "List warehouse actions",
			Examples: []string{
				"fakturownia warehouse-action list --json",
				"fakturownia warehouse-action list --warehouse-id 1 --kind mm --product-id 7 --json",
				"fakturownia warehouse-action list --warehouse-document-id 15 --columns id,kind,product_id,quantity,warehouse_document_id",
				"fakturownia warehouse-action list --page 2 --per-page 25 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "page", Type: "int", Description: "Requested result page", Default: "1"},
				{Name: "per-page", Type: "int", Description: "Requested result count per page", Default: "25"},
				{Name: "warehouse-id", Type: "string", Description: "Filter by warehouse ID"},
				{Name: "kind", Type: "string", Description: "Filter by warehouse action kind"},
				{Name: "product-id", Type: "string", Description: "Filter by product ID"},
				{Name: "date-from", Type: "string", Description: "Filter actions created on or after a date such as 2026-04-01"},
				{Name: "date-to", Type: "string", Description: "Filter actions created on or before a date such as 2026-04-15"},
				{Name: "from-warehouse-document", Type: "string", Description: "Filter actions linked from a warehouse document ID"},
				{Name: "to-warehouse-document", Type: "string", Description: "Filter actions linked to a warehouse document ID"},
				{Name: "warehouse-document-id", Type: "string", Description: "Filter by warehouse document ID"},
			},
			DataPrototype: []map[string]any{},
			Output:        warehouseActionListOutputSpec(),
		},
		{
			Noun:  "doctor",
			Verb:  "run",
			Use:   "run",
			Short: "Validate local auth and API reachability",
			Examples: []string{
				"fakturownia doctor run",
				"fakturownia doctor run --check-release-integrity --json",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			LocalFlags: []FlagSpec{
				{Name: "check-release-integrity", Type: "bool", Description: "Verify the running binary against the published release checksum when available", Default: "false"},
			},
			DataPrototype: doctor.Report{},
		},
		{
			Noun:  "self",
			Verb:  "update",
			Use:   "update",
			Short: "Replace the running binary with a GitHub Release build",
			Examples: []string{
				"fakturownia self update",
				"fakturownia self update --version v0.2.0",
				"fakturownia self update --dry-run --json",
			},
			EnvVars:      nil,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: false,
			Mutating:     true,
			LocalFlags: []FlagSpec{
				{Name: "version", Type: "string", Description: "Release tag to install, or latest when omitted"},
			},
			DataPrototype: selfupdate.UpdateResult{},
		},
		{
			Noun:          "schema",
			Verb:          "list",
			Use:           "list",
			Short:         "List all supported commands",
			Examples:      []string{"fakturownia schema list", "fakturownia schema list --json"},
			EnvVars:       nil,
			OutputModes:   []string{"human", "json"},
			ExitCodes:     exitCodes,
			RawSupported:  false,
			DataPrototype: []SchemaSummary{},
		},
		{
			Noun:          "schema",
			Verb:          "<noun> <verb>",
			Use:           "<noun> <verb>",
			Short:         "Describe a command as JSON schema",
			Examples:      []string{"fakturownia schema invoice list", "fakturownia schema auth login --json"},
			EnvVars:       nil,
			OutputModes:   []string{"human", "json"},
			ExitCodes:     exitCodes,
			RawSupported:  false,
			DataPrototype: CommandSchema{},
		},
	}
}

func FindCommand(noun, verb string) (CommandSpec, bool) {
	for _, spec := range Registry() {
		if spec.Noun == noun && spec.Verb == verb {
			return spec, true
		}
	}
	return CommandSpec{}, false
}

func SchemaSummaries() []SchemaSummary {
	specs := Registry()
	summaries := make([]SchemaSummary, 0, len(specs))
	for _, spec := range specs {
		summaries = append(summaries, SchemaSummary{
			Noun:    spec.Noun,
			Verb:    spec.Verb,
			Use:     spec.Use,
			Summary: spec.Short,
		})
	}
	return summaries
}

func BuildCommandSchema(spec CommandSpec) (CommandSchema, error) {
	dataSchema, err := buildOutputDataSchema(spec.Output)
	if err != nil {
		return CommandSchema{}, err
	}
	if dataSchema == nil {
		dataSchema, err = reflectSchema(spec.DataPrototype)
	}
	if err != nil {
		return CommandSchema{}, err
	}
	requestBodySchema, err := buildRequestBodySchema(spec.RequestBody)
	if err != nil {
		return CommandSchema{}, err
	}
	errorSchema, err := reflectSchema(output.ErrorDetail{})
	if err != nil {
		return CommandSchema{}, err
	}
	warningSchema, err := reflectSchema(output.WarningDetail{})
	if err != nil {
		return CommandSchema{}, err
	}
	metaSchema, err := reflectSchema(output.Meta{})
	if err != nil {
		return CommandSchema{}, err
	}

	flags := append([]FlagSpec{}, GlobalFlags()...)
	flags = append(flags, spec.LocalFlags...)
	return CommandSchema{
		SchemaVersion:     output.SchemaVersion,
		Command:           spec.Noun + " " + spec.Verb,
		Use:               spec.Use,
		Summary:           spec.Short,
		Flags:             flags,
		EnvVars:           spec.EnvVars,
		OutputModes:       spec.OutputModes,
		ExitCodes:         spec.ExitCodes,
		RawSupported:      spec.RawSupported,
		Examples:          spec.Examples,
		DataSchema:        dataSchema,
		RequestBody:       cloneRequestBodySpec(spec.RequestBody),
		RequestBodySchema: requestBodySchema,
		EnvelopeSchema:    envelopeSchema(dataSchema, errorSchema, warningSchema, metaSchema),
		Output:            spec.Output,
	}, nil
}

func BuildLongDescription(spec CommandSpec) string {
	var lines []string
	lines = append(lines, spec.Short)
	lines = append(lines, "")
	lines = append(lines, "Usage")
	lines = append(lines, fmt.Sprintf("  fakturownia %s %s", spec.Noun, spec.Use))
	lines = append(lines, "")

	flags := append([]FlagSpec{}, spec.LocalFlags...)
	if len(flags) > 0 {
		lines = append(lines, "Required Flags")
		for _, flag := range flags {
			if flag.Required {
				lines = append(lines, fmt.Sprintf("  --%s (%s): %s", flag.Name, flag.Type, flag.Description))
			}
		}
		lines = append(lines, "")
		lines = append(lines, "Optional Flags")
		for _, flag := range flags {
			if !flag.Required {
				lines = append(lines, formatFlag(flag))
			}
		}
		lines = append(lines, "")
	}

	lines = append(lines, "Global Flags")
	for _, flag := range GlobalFlags() {
		lines = append(lines, formatFlag(flag))
	}
	lines = append(lines, "")

	if len(spec.EnvVars) > 0 {
		lines = append(lines, "Environment Variables")
		for _, env := range spec.EnvVars {
			lines = append(lines, fmt.Sprintf("  %s: %s", env.Name, env.Description))
		}
		lines = append(lines, "")
	}

	lines = append(lines, "Output Modes")
	lines = append(lines, fmt.Sprintf("  %s", strings.Join(spec.OutputModes, ", ")))
	if spec.RawSupported {
		lines = append(lines, "  raw: supported")
	} else {
		lines = append(lines, "  raw: unsupported")
	}
	lines = append(lines, "")

	lines = append(lines, "Exit Codes")
	for _, code := range spec.ExitCodes {
		lines = append(lines, fmt.Sprintf("  %d: %s", code.Code, code.Description))
	}
	lines = append(lines, "")

	lines = append(lines, "Examples")
	for _, example := range spec.Examples {
		lines = append(lines, "  "+example)
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func reflectSchema(prototype any) (map[string]any, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: true,
	}
	schema := reflector.Reflect(prototype)
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return generic, nil
}

func envelopeSchema(dataSchema, errorSchema, warningSchema, metaSchema map[string]any) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"schema_version": map[string]any{"type": "string"},
			"status":         map[string]any{"type": "string", "enum": []string{"success", "error"}},
			"data":           dataSchema,
			"errors": map[string]any{
				"type":  "array",
				"items": errorSchema,
			},
			"warnings": map[string]any{
				"type":  "array",
				"items": warningSchema,
			},
			"meta": metaSchema,
		},
		"required": []string{"schema_version", "status", "data", "errors", "warnings", "meta"},
	}
}

func formatFlag(flag FlagSpec) string {
	parts := []string{fmt.Sprintf("  --%s (%s): %s", flag.Name, flag.Type, flag.Description)}
	if flag.Default != "" {
		parts = append(parts, fmt.Sprintf("default=%s", flag.Default))
	}
	if len(flag.Enum) > 0 {
		enum := append([]string{}, flag.Enum...)
		slices.Sort(enum)
		parts = append(parts, fmt.Sprintf("enum=%s", strings.Join(enum, ",")))
	}
	return strings.Join(parts, " ")
}
