package spec

func clientListOutputSpec() *OutputSpec {
	spec := clientBaseOutputSpec("array", []string{"client list", "client get"})
	spec.KnownFields = append([]OutputFieldSpec{}, spec.KnownFields...)
	return spec
}

func clientGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"client get"}
	}
	return clientBaseOutputSpec("object", commands)
}

func clientRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "client",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax: "dot_bracket",
		KnownFields: clientRequestFields(),
		Notes: []string{
			"the CLI accepts the inner client object, then wraps it in the upstream {\"client\": ...} envelope",
			"known_fields is curated from the upstream README and is not exhaustive",
		},
	}
}

func clientBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "name", "tax_no", "email", "city", "country"},
		Notes: []string{
			"known_fields is curated from the upstream README and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
		},
		KnownFields: clientKnownOutputFields(commands),
	}
}

func clientKnownOutputFields(commands []string) []OutputFieldSpec {
	return []OutputFieldSpec{
		{Path: "id", Type: "integer", Description: "Client ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie wybranego klienta po ID"},
		{Path: "name", Type: "string", Description: "Client display name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "shortcut", Type: "string", Description: "Short client name", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "tax_no_kind", Type: "string", Description: "Tax identifier kind", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "tax_no", Type: "string", Description: "Tax identifier value", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "register_number", Type: "string", Description: "REGON or register number", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "accounting_id", Type: "string", Description: "Accounting-system identifier", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "post_code", Type: "string", Description: "Postal code", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "city", Type: "string", Description: "City", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "street", Type: "string", Description: "Street address", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "country", Type: "string", Description: "Country code", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "use_delivery_address", Type: "string", Description: "Use alternate delivery address flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta", EnumValues: []string{"1", "0"}},
		{Path: "delivery_address", Type: "string", Description: "Alternate delivery address", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "first_name", Type: "string", Description: "First name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "last_name", Type: "string", Description: "Last name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "email", Type: "string", Description: "Email address", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "phone", Type: "string", Description: "Phone number", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta"},
		{Path: "mobile_phone", Type: "string", Description: "Mobile phone number", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "www", Type: "string", Description: "Website URL", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "fax", Type: "string", Description: "Fax number", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "note", Type: "string", Description: "Additional note", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "tag_list[]", Type: "string", Description: "Client tags", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "company", Type: "string", Description: "Company flag", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola klienta", EnumValues: []string{"1", "0"}},
		{Path: "kind", Type: "string", Description: "Client kind", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta", EnumValues: []string{"buyer", "seller", "both"}},
		{Path: "category_id", Type: "integer", Description: "Category ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "bank", Type: "string", Description: "Bank name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "bank_account", Type: "string", Description: "Bank account number", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "discount", Type: "number", Description: "Default discount percentage", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "default_tax", Type: "number", Description: "Default tax rate", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "price_list_id", Type: "integer", Description: "Default price list ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "payment_to_kind", Type: "string", Description: "Default payment term", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "default_payment_type", Type: "string", Description: "Default payment type", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta", EnumValues: []string{"transfer", "card", "cash", "barter", "cheque", "bill_of_exchange", "cash_on_delivery", "compensation", "letter_of_credit", "payu", "paypal", "off"}},
		{Path: "disable_auto_reminders", Type: "string", Description: "Disable automatic reminders flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta", EnumValues: []string{"1", "0"}},
		{Path: "person", Type: "string", Description: "Contact person", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "buyer_id", Type: "integer", Description: "Linked buyer ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "mass_payment_code", Type: "string", Description: "Mass payment code", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta"},
		{Path: "external_id", Type: "string", Description: "External client ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie wybranego klienta po zewnętrznym ID"},
		{Path: "tp_client_connection", Type: "string", Description: "Related-entity TP flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola klienta", EnumValues: []string{"1", "0"}},
	}
}

func clientRequestFields() []RequestFieldSpec {
	outputFields := clientKnownOutputFields([]string{"client create", "client update"})
	fields := make([]RequestFieldSpec, 0, len(outputFields))
	for _, field := range outputFields {
		fields = append(fields, RequestFieldSpec{
			Path:          field.Path,
			Type:          field.Type,
			Description:   field.Description,
			EnumValues:    append([]string{}, field.EnumValues...),
			SourceSection: field.SourceSection,
		})
	}
	return fields
}
