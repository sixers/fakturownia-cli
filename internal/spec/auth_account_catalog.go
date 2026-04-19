package spec

func authExchangeOutputSpec() *OutputSpec {
	commands := []string{"auth exchange"}
	return &OutputSpec{
		Shape:      "object",
		OpenEnded:  false,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		Notes: []string{
			"structured output is sanitized and does not expose the returned api_token; use --raw for the exact upstream response",
		},
		KnownFields: []OutputFieldSpec{
			{Path: "login", Type: "string", Description: "Resolved user login", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Logowanie i pobranie danych przez API"},
			{Path: "email", Type: "string", Description: "Resolved user email", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Logowanie i pobranie danych przez API"},
			{Path: "prefix", Type: "string", Description: "Account prefix", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Logowanie i pobranie danych przez API"},
			{Path: "url", Type: "string", Description: "Account URL", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Logowanie i pobranie danych przez API"},
			{Path: "first_name", Type: "string", Description: "User first name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Logowanie i pobranie danych przez API"},
			{Path: "last_name", Type: "string", Description: "User last name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Logowanie i pobranie danych przez API"},
			{Path: "api_token_present", Type: "boolean", Description: "Whether the upstream response included an API token", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Logowanie i pobranie danych przez API"},
			{Path: "saved_profile", Type: "string", Description: "Persisted profile name", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "CLI"},
			{Path: "token_stored", Type: "boolean", Description: "Whether the token was stored in the configured credential store", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "CLI"},
			{Path: "config_path", Type: "string", Description: "Config file path used for profile persistence", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "CLI"},
		},
	}
}

func userCreateRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "user",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax: "dot_bracket",
		KnownFields: []RequestFieldSpec{
			{Path: "invite", Type: "boolean", Description: "Invite an existing user instead of creating one", SourceSection: "Dodawanie użytkowników"},
			{Path: "email", Type: "string", Description: "User email address", SourceSection: "Dodawanie użytkowników"},
			{Path: "password", Type: "string", Description: "User password when invite=false", SourceSection: "Dodawanie użytkowników"},
			{Path: "role", Type: "string", Description: "User role such as member, admin, accountant, or a custom role ID", SourceSection: "Dodawanie użytkowników"},
			{Path: "department_ids[]", Type: "array<integer>", Description: "Department IDs available to the user", SourceSection: "Dodawanie użytkowników"},
		},
		Notes: []string{
			"the CLI accepts the inner user object, then wraps it in the upstream {\"user\": ...} envelope",
			"pass --integration-token separately; the active profile provides api_token automatically",
			"known_fields is curated from the upstream README and is not exhaustive",
		},
	}
}

func accountCreateOutputSpec() *OutputSpec {
	return accountBaseOutputSpec([]string{"account create"}, true)
}

func accountGetOutputSpec() *OutputSpec {
	return accountBaseOutputSpec([]string{"account get"}, false)
}

func accountDeleteOutputSpec() *OutputSpec {
	return &OutputSpec{
		Shape:      "object",
		OpenEnded:  false,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		KnownFields: []OutputFieldSpec{
			{Path: "code", Type: "string", Description: "Result code", Projectable: true, Selectable: true, Commands: []string{"account delete"}, Presence: "common", SourceSection: "Usuwanie konta"},
			{Path: "message", Type: "string", Description: "Result message", Projectable: true, Selectable: true, Commands: []string{"account delete"}, Presence: "common", SourceSection: "Usuwanie konta"},
		},
	}
}

func accountUnlinkOutputSpec() *OutputSpec {
	return &OutputSpec{
		Shape:      "object",
		OpenEnded:  false,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		KnownFields: []OutputFieldSpec{
			{Path: "code", Type: "string", Description: "Result code", Projectable: true, Selectable: true, Commands: []string{"account unlink"}, Presence: "common", SourceSection: "Odpięcie kont systemowych"},
			{Path: "message", Type: "string", Description: "Result message", Projectable: true, Selectable: true, Commands: []string{"account unlink"}, Presence: "common", SourceSection: "Odpięcie kont systemowych"},
			{Path: "result.unlinked[]", Type: "array<string>", Description: "Prefixes successfully unlinked", Projectable: true, Selectable: true, Commands: []string{"account unlink"}, Presence: "conditional", SourceSection: "Odpięcie kont systemowych"},
			{Path: "result.not_unlinked[]", Type: "array<string>", Description: "Prefixes that could not be unlinked", Projectable: true, Selectable: true, Commands: []string{"account unlink"}, Presence: "conditional", SourceSection: "Odpięcie kont systemowych"},
		},
	}
}

func accountBaseOutputSpec(commands []string, includeSaveFields bool) *OutputSpec {
	fields := []OutputFieldSpec{
		{Path: "prefix", Type: "string", Description: "Account prefix", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Konta Systemowe"},
		{Path: "url", Type: "string", Description: "Account URL", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Konta Systemowe"},
		{Path: "login", Type: "string", Description: "Account owner login", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Konta Systemowe"},
		{Path: "email", Type: "string", Description: "Account owner email", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Konta Systemowe"},
		{Path: "api_token_present", Type: "boolean", Description: "Whether the upstream response included an API token", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Konta Systemowe"},
	}
	if includeSaveFields {
		fields = append(fields,
			OutputFieldSpec{Path: "saved_profile", Type: "string", Description: "Persisted profile name", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "CLI"},
			OutputFieldSpec{Path: "token_stored", Type: "boolean", Description: "Whether the returned token was stored", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "CLI"},
			OutputFieldSpec{Path: "config_path", Type: "string", Description: "Config file path used for persistence", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "CLI"},
		)
	}
	return &OutputSpec{
		Shape:      "object",
		OpenEnded:  false,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		Notes: []string{
			"structured output is sanitized and does not expose the returned api_token; use --raw for the exact upstream response",
		},
		KnownFields: fields,
	}
}

func accountCreateRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax: "dot_bracket",
		KnownFields: []RequestFieldSpec{
			{Path: "account.prefix", Type: "string", Description: "Requested account prefix", SourceSection: "Zakładanie konta"},
			{Path: "account.lang", Type: "string", Description: "Account language", SourceSection: "Zakładanie konta"},
			{Path: "account.integration_fast_login", Type: "boolean", Description: "Enable partner fast login", SourceSection: "Zakładanie konta"},
			{Path: "account.integration_logout_url", Type: "string", Description: "Partner logout redirect URL", SourceSection: "Zakładanie konta"},
			{Path: "user.login", Type: "string", Description: "Account owner login", SourceSection: "Zakładanie konta"},
			{Path: "user.email", Type: "string", Description: "Account owner email", SourceSection: "Zakładanie konta"},
			{Path: "user.password", Type: "string", Description: "Account owner password", SourceSection: "Zakładanie konta"},
			{Path: "user.from_partner", Type: "string", Description: "Partner code", SourceSection: "Zakładanie konta"},
			{Path: "company.name", Type: "string", Description: "Company name", SourceSection: "Zakładanie konta"},
			{Path: "company.tax_no", Type: "string", Description: "Company tax number", SourceSection: "Zakładanie konta"},
			{Path: "company.post_code", Type: "string", Description: "Company postal code", SourceSection: "Zakładanie konta"},
			{Path: "company.city", Type: "string", Description: "Company city", SourceSection: "Zakładanie konta"},
			{Path: "company.street", Type: "string", Description: "Company street", SourceSection: "Zakładanie konta"},
			{Path: "company.person", Type: "string", Description: "Company contact person", SourceSection: "Zakładanie konta"},
			{Path: "company.bank", Type: "string", Description: "Company bank", SourceSection: "Zakładanie konta"},
			{Path: "company.bank_account", Type: "string", Description: "Company bank account", SourceSection: "Zakładanie konta"},
			{Path: "integration_token", Type: "string", Description: "Integration token", SourceSection: "Zakładanie konta"},
		},
		Notes: []string{
			"account create accepts the full top-level request object described in the upstream README",
			"the active profile provides api_token automatically; do not include it in --input",
			"known_fields is curated from the upstream README and is not exhaustive",
		},
	}
}
