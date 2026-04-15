package spec

func bankAccountListOutputSpec() *OutputSpec {
	return bankAccountBaseOutputSpec("array", []string{"bank-account list"})
}

func bankAccountGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"bank-account get", "bank-account create", "bank-account update"}
	}
	return bankAccountBaseOutputSpec("object", commands)
}

func bankAccountRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "bank_account",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "bank_accounts",
			URL:    fakturowniaBankAccountsURL,
		},
		PathSyntax: "dot_bracket",
		KnownFields: []RequestFieldSpec{
			{Path: "bank", Type: "string", Description: "Bank display name used by the write examples", SourceSection: "Tworzenie rachunku bankowego"},
			{Path: "bank_name", Type: "string", Description: "Bank name returned by the API", SourceSection: "Pobranie listy rachunków bankowych"},
			{Path: "bank_swift", Type: "string", Description: "Bank SWIFT or BIC code", SourceSection: "Tworzenie rachunku bankowego"},
			{Path: "bank_account", Type: "string", Description: "Bank account number", SourceSection: "Tworzenie rachunku bankowego"},
			{Path: "bank_account_currency", Type: "string", Description: "Bank account currency", SourceSection: "Tworzenie rachunku bankowego"},
			{Path: "default", Type: "boolean", Description: "Mark this bank account as the default one", SourceSection: "Tworzenie rachunku bankowego"},
			{Path: "bank_account_version_departments[]", Type: "array<object>", Description: "Per-department bank-account visibility settings", SourceSection: "Tworzenie rachunku bankowego z przypisaniem do działu"},
			{Path: "bank_account_version_departments[].department_id", Type: "integer", Description: "Department ID that should use this bank account", SourceSection: "Tworzenie rachunku bankowego z przypisaniem do działu"},
			{Path: "bank_account_version_departments[].remove", Type: "boolean", Description: "Remove the bank-account assignment from the department", SourceSection: "Edycja rachunku bankowego z przypisaniem do działu"},
			{Path: "bank_account_version_departments[].main_on_department", Type: "boolean", Description: "Use this bank account as the department default", SourceSection: "Tworzenie rachunku bankowego z przypisaniem do działu"},
			{Path: "bank_account_version_departments[].show_on_invoice", Type: "boolean", Description: "Show this bank account on department invoices", SourceSection: "Tworzenie rachunku bankowego z przypisaniem do działu"},
		},
		Notes: []string{
			"the CLI accepts the inner bank_account object, then wraps it in the upstream {\"bank_account\": ...} envelope",
			"known_fields is curated from API_RACHUNKI_BANKOWE.md and is not exhaustive",
			"the upstream addendum uses `bank` in create and update examples while returned objects expose `bank_name`; both are surfaced for discovery",
		},
	}
}

func bankAccountBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "bank_accounts",
			URL:    fakturowniaBankAccountsURL,
		},
		DefaultColumns: []string{"id", "bank_name", "bank_account", "bank_account_currency", "default"},
		Notes: []string{
			"known_fields is curated from API_RACHUNKI_BANKOWE.md and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
			"the upstream addendum uses `bank` in write examples while returned objects expose `bank_name`; both are surfaced in schema discovery where appropriate",
		},
		KnownFields: []OutputFieldSpec{
			{Path: "id", Type: "integer", Description: "Bank account ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie pojedycznego rachunku bankowego po ID"},
			{Path: "bank", Type: "string", Description: "Bank display name used by write examples", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Tworzenie rachunku bankowego"},
			{Path: "bank_name", Type: "string", Description: "Bank name returned by the API", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie listy rachunków bankowych"},
			{Path: "bank_swift", Type: "string", Description: "Bank SWIFT or BIC code", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie listy rachunków bankowych"},
			{Path: "bank_account", Type: "string", Description: "Bank account number", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie listy rachunków bankowych"},
			{Path: "bank_account_currency", Type: "string", Description: "Bank account currency", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie listy rachunków bankowych"},
			{Path: "default", Type: "boolean", Description: "Whether this bank account is the default one", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie listy rachunków bankowych"},
			{Path: "bank_account_version_departments[]", Type: "array<object>", Description: "Per-department bank-account visibility settings", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", SourceSection: "Pobranie pojedycznego rachunku bankowego po ID"},
			{Path: "bank_account_version_departments[].department_id", Type: "integer", Description: "Department ID that uses this bank account", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie pojedycznego rachunku bankowego po ID"},
			{Path: "bank_account_version_departments[].main_on_department", Type: "boolean", Description: "Whether the bank account is the main one on that department", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie pojedycznego rachunku bankowego po ID"},
			{Path: "bank_account_version_departments[].show_on_invoice", Type: "boolean", Description: "Whether the bank account is shown on department invoices", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie pojedycznego rachunku bankowego po ID"},
		},
	}
}
