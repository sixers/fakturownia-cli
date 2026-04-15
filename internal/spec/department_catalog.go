package spec

func departmentListOutputSpec() *OutputSpec {
	return departmentBaseOutputSpec("array", []string{"department list"})
}

func departmentGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"department get", "department create", "department update", "department set-logo"}
	}
	return departmentBaseOutputSpec("object", commands)
}

func departmentRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "department",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		AdditionalCatalogBases: []*CatalogBasis{
			{
				Source: "bank_accounts",
				URL:    fakturowniaBankAccountsURL,
			},
		},
		PathSyntax: "dot_bracket",
		KnownFields: []RequestFieldSpec{
			{Path: "name", Type: "string", Description: "Department name", SourceSection: "Dodanie nowego działu"},
			{Path: "shortcut", Type: "string", Description: "Department shortcut", SourceSection: "Dodanie nowego działu"},
			{Path: "tax_no", Type: "string", Description: "Department tax number", SourceSection: "Dodanie nowego działu"},
			{Path: "bank_account_id", Type: "integer", Description: "Default department bank account ID", SourceSection: "Tworzenie nowej faktury z własnym kodem do płatności masowych"},
			{Path: "show_bank_account_on_invoice", Type: "string", Description: "Show the department bank account on invoices", SourceSection: "Tworzenie nowej faktury z własnym kodem do płatności masowych", EnumValues: []string{"1", "0"}},
			{Path: "mass_payment_pattern", Type: "string", Description: "Department mass-payment pattern", SourceSection: "Tworzenie nowej faktury z własnym kodem do płatności masowych"},
		},
		Notes: []string{
			"the CLI accepts the inner department object, then wraps it in the upstream {\"department\": ...} envelope",
			"known_fields is curated from the upstream README plus API_RACHUNKI_BANKOWE.md and is not exhaustive",
		},
	}
}

func departmentBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		AdditionalCatalogBases: []*CatalogBasis{
			{
				Source: "bank_accounts",
				URL:    fakturowniaBankAccountsURL,
			},
		},
		DefaultColumns: []string{"id", "name", "shortcut", "tax_no"},
		Notes: []string{
			"known_fields is curated from the upstream README plus API_RACHUNKI_BANKOWE.md and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
		},
		KnownFields: []OutputFieldSpec{
			{Path: "id", Type: "integer", Description: "Department ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie pojedycznego działu po ID"},
			{Path: "name", Type: "string", Description: "Department name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodanie nowego działu"},
			{Path: "shortcut", Type: "string", Description: "Department shortcut", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie nowego działu"},
			{Path: "tax_no", Type: "string", Description: "Department tax number", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie nowego działu"},
			{Path: "bank_account_id", Type: "integer", Description: "Default department bank account ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Tworzenie nowej faktury z własnym kodem do płatności masowych"},
			{Path: "show_bank_account_on_invoice", Type: "string", Description: "Show the department bank account on invoices", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Tworzenie nowej faktury z własnym kodem do płatności masowych", EnumValues: []string{"1", "0"}},
			{Path: "mass_payment_pattern", Type: "string", Description: "Department mass-payment pattern", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Tworzenie nowej faktury z własnym kodem do płatności masowych"},
		},
	}
}
