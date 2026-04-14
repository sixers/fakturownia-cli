package spec

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/invopop/jsonschema"

	"github.com/sixers/fakturownia-cli/internal/auth"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/output"
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
	DataPrototype any
}

type SchemaSummary struct {
	Noun    string `json:"noun"`
	Verb    string `json:"verb"`
	Use     string `json:"use"`
	Summary string `json:"summary"`
}

type CommandSchema struct {
	SchemaVersion  string         `json:"schema_version"`
	Command        string         `json:"command"`
	Use            string         `json:"use"`
	Summary        string         `json:"summary"`
	Flags          []FlagSpec     `json:"flags"`
	EnvVars        []EnvVarSpec   `json:"env_vars"`
	OutputModes    []string       `json:"output_modes"`
	ExitCodes      []ExitCodeSpec `json:"exit_codes"`
	RawSupported   bool           `json:"raw_supported"`
	Examples       []string       `json:"examples"`
	DataSchema     map[string]any `json:"data_schema"`
	EnvelopeSchema map[string]any `json:"envelope_schema"`
}

func GlobalFlags() []FlagSpec {
	return []FlagSpec{
		{Name: "profile", Type: "string", Description: "Select a named profile", Default: ""},
		{Name: "json", Type: "bool", Description: "Alias for --output json", Default: "false"},
		{Name: "output", Type: "string", Description: "Output format", Default: "human", Enum: []string{"human", "json"}},
		{Name: "quiet", Type: "bool", Description: "Emit bare values when exactly one field or column remains", Default: "false"},
		{Name: "fields", Type: "string[]", Description: "Project JSON envelope data fields", Repeatable: true},
		{Name: "columns", Type: "string[]", Description: "Select human table columns", Repeatable: true},
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
			Noun:  "invoice",
			Verb:  "list",
			Use:   "list",
			Short: "List invoices",
			Examples: []string{
				"fakturownia invoice list --json",
				"fakturownia invoice list --period this_month --columns id,number,buyer_name,price_gross",
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
				{Name: "kind", Type: "string[]", Description: "Filter by invoice kind", Repeatable: true},
				{Name: "search-date-type", Type: "string", Description: "Date field to search by", Enum: []string{"issue_date", "paid_date", "transaction_date"}},
				{Name: "order", Type: "string", Description: "Sort order"},
				{Name: "income", Type: "string", Description: "Income selector", Enum: []string{"yes", "no"}},
			},
			DataPrototype: []map[string]any{},
		},
		{
			Noun:  "invoice",
			Verb:  "get",
			Use:   "get --id ID",
			Short: "Fetch a single invoice by ID",
			Examples: []string{
				"fakturownia invoice get --id 123",
				"fakturownia invoice get --id 123 --fields id,number,status --json",
				"fakturownia invoice get --id 123 --raw",
			},
			EnvVars:      env,
			OutputModes:  []string{"human", "json"},
			ExitCodes:    exitCodes,
			RawSupported: true,
			LocalFlags: []FlagSpec{
				{Name: "id", Type: "string", Description: "Invoice ID", Required: true},
			},
			DataPrototype: map[string]any{},
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
	dataSchema, err := reflectSchema(spec.DataPrototype)
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
		SchemaVersion:  output.SchemaVersion,
		Command:        spec.Noun + " " + spec.Verb,
		Use:            spec.Use,
		Summary:        spec.Short,
		Flags:          flags,
		EnvVars:        spec.EnvVars,
		OutputModes:    spec.OutputModes,
		ExitCodes:      spec.ExitCodes,
		RawSupported:   spec.RawSupported,
		Examples:       spec.Examples,
		DataSchema:     dataSchema,
		EnvelopeSchema: envelopeSchema(dataSchema, errorSchema, warningSchema, metaSchema),
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
