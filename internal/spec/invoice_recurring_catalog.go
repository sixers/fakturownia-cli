package spec

func invoiceCreateRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "invoice",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		AdditionalCatalogBases: []*CatalogBasis{
			{
				Source: "ksef",
				URL:    fakturowniaKSeFURL,
			},
			{
				Source: "bank_accounts",
				URL:    fakturowniaBankAccountsURL,
			},
		},
		PathSyntax:  "dot_bracket",
		KnownFields: invoiceRequestFields(),
		Notes: []string{
			"the CLI accepts the inner invoice object, then wraps it in the upstream {\"invoice\": ...} envelope",
			"companion CLI flags `--identify-oss`, `--fill-default-descriptions`, `--correction-positions`, and `--gov-save-and-send` are applied outside the inner invoice object",
			"known_fields is curated from the upstream README and is not exhaustive",
		},
	}
}

func invoiceUpdateRequestBodySpec() *RequestBodySpec {
	spec := invoiceCreateRequestBodySpec()
	spec.Notes = append(append([]string{}, spec.Notes...),
		"invoice position updates use `positions[].id`; passing `positions[]._destroy` removes a position and omitting `id` appends a new one",
	)
	return spec
}

func invoiceRequestFields() []RequestFieldSpec {
	return []RequestFieldSpec{
		{Path: "kind", Type: "string", Description: "Invoice kind", SourceSection: "Dodanie nowej faktury", EnumValues: []string{"vat", "proforma", "bill", "receipt", "advance", "final", "correction", "invoice_other", "vat_margin", "kp", "kw", "estimate", "vat_mp", "vat_rr", "correction_note", "accounting_note"}},
		{Path: "number", Type: "string", Description: "Invoice number or null for auto-numbering", SourceSection: "Dodanie nowej faktury"},
		{Path: "sell_date", Type: "string", Description: "Invoice sale date", SourceSection: "Dodanie nowej faktury"},
		{Path: "issue_date", Type: "string", Description: "Invoice issue date", SourceSection: "Dodanie nowej faktury"},
		{Path: "payment_to", Type: "string", Description: "Invoice payment due date", SourceSection: "Dodanie nowej faktury"},
		{Path: "payment_to_kind", Type: "integer", Description: "Relative payment term in days", SourceSection: "Dodanie nowej faktury - minimalna wersja"},
		{Path: "payment_type", Type: "string", Description: "Payment type", SourceSection: "Dodanie nowej faktury", EnumValues: []string{"transfer", "card", "cash", "barter", "cheque", "bill_of_exchange", "cash_on_delivery", "compensation", "letter_of_credit", "payu", "paypal", "off"}},
		{Path: "currency", Type: "string", Description: "Invoice currency", SourceSection: "Dodanie nowej faktury"},
		{Path: "bank_account_id", Type: "integer", Description: "Referenced bank account ID used on the invoice", SourceSection: "Tworzenie nowej faktury z istniejącym rachunkiem bankowym"},
		{Path: "buyer_mass_payment_code", Type: "string", Description: "Custom mass-payment code printed for the buyer", SourceSection: "Tworzenie nowej faktury z własnym kodem do płatności masowych"},
		{Path: "bank_accounts[]", Type: "array<object>", Description: "Bank accounts embedded directly into the invoice payload", SourceSection: "Tworzenie nowej faktury z rachunkiem bankowym"},
		{Path: "bank_accounts[].bank_name", Type: "string", Description: "Embedded bank name", SourceSection: "Tworzenie nowej faktury z rachunkiem bankowym"},
		{Path: "bank_accounts[].bank_account_number", Type: "string", Description: "Embedded bank account number", SourceSection: "Tworzenie nowej faktury z rachunkiem bankowym"},
		{Path: "bank_accounts[].bank_currency", Type: "string", Description: "Embedded bank account currency", SourceSection: "Tworzenie nowej faktury z rachunkiem bankowym"},
		{Path: "bank_accounts[].bank_swift", Type: "string", Description: "Embedded bank SWIFT or BIC code", SourceSection: "Tworzenie nowej faktury z rachunkiem bankowym"},
		{Path: "seller_name", Type: "string", Description: "Seller display name", SourceSection: "Dodanie nowej faktury"},
		{Path: "seller_tax_no", Type: "string", Description: "Seller tax number", SourceSection: "Dodanie nowej faktury"},
		{Path: "seller_tax_no_kind", Type: "string", Description: "Seller tax identifier kind", SourceSection: "Rodzaj identyfikatora podatkowego (tax_no_kind)", EnumValues: []string{"", "nip_ue", "other", "empty", "nip_with_id"}},
		{Path: "seller_street", Type: "string", Description: "Seller street and number", SourceSection: "Zmiany w API przy tworzeniu faktur"},
		{Path: "seller_post_code", Type: "string", Description: "Seller postal code", SourceSection: "Zmiany w API przy tworzeniu faktur"},
		{Path: "seller_city", Type: "string", Description: "Seller city", SourceSection: "Zmiany w API przy tworzeniu faktur"},
		{Path: "seller_country", Type: "string", Description: "Seller country code", SourceSection: "Dodanie faktury OSS"},
		{Path: "buyer_name", Type: "string", Description: "Buyer display name", SourceSection: "Dodanie nowej faktury"},
		{Path: "buyer_email", Type: "string", Description: "Buyer email address", SourceSection: "Dodanie nowej faktury"},
		{Path: "buyer_tax_no", Type: "string", Description: "Buyer tax number", SourceSection: "Dodanie nowej faktury"},
		{Path: "buyer_tax_no_kind", Type: "string", Description: "Buyer tax identifier kind", SourceSection: "Rodzaj identyfikatora podatkowego (tax_no_kind)", EnumValues: []string{"", "nip_ue", "other", "empty", "nip_with_id"}},
		{Path: "buyer_company", Type: "boolean", Description: "Whether the buyer is a company", SourceSection: "Zmiany w API przy tworzeniu faktur"},
		{Path: "buyer_first_name", Type: "string", Description: "Buyer first name for B2C invoices", SourceSection: "Nabywca - osoba fizyczna"},
		{Path: "buyer_last_name", Type: "string", Description: "Buyer last name for B2C invoices", SourceSection: "Nabywca - osoba fizyczna"},
		{Path: "buyer_country", Type: "string", Description: "Buyer country code", SourceSection: "Dodanie faktury OSS"},
		{Path: "buyer_post_code", Type: "string", Description: "Buyer postal code override", SourceSection: "Dodanie nowej faktury"},
		{Path: "buyer_city", Type: "string", Description: "Buyer city override", SourceSection: "Dodanie nowej faktury"},
		{Path: "buyer_street", Type: "string", Description: "Buyer street override", SourceSection: "Dodanie nowej faktury"},
		{Path: "buyer_override", Type: "boolean", Description: "Update selected buyer fields on the linked client card", SourceSection: "Dodanie nowej faktury"},
		{Path: "client_id", Type: "integer", Description: "Referenced client ID", SourceSection: "Dodanie nowej faktury - minimalna wersja"},
		{Path: "department_id", Type: "integer", Description: "Referenced seller department ID", SourceSection: "Dodanie nowej faktury - minimalna wersja"},
		{Path: "recipient_id", Type: "integer", Description: "Referenced recipient ID", SourceSection: "Dodanie nowej faktury - minimalna wersja"},
		{Path: "use_oss", Type: "boolean", Description: "Mark the invoice as OSS", SourceSection: "Dodanie faktury OSS"},
		{Path: "income", Type: "string", Description: "Income selector; use 0 for expense invoices", SourceSection: "Dodanie faktury kosztowej", EnumValues: []string{"1", "0"}},
		{Path: "copy_invoice_from", Type: "integer", Description: "Copy invoice, order, or proforma from another document", SourceSection: "Dodanie nowej faktury – dokumentu podobnego do faktury o podanym ID (copy_invoice_from)"},
		{Path: "advance_creation_mode", Type: "string", Description: "Advance invoice creation mode", SourceSection: "Dodanie faktury zaliczkowej na podstawie zamówienia – % pełnej kwoty", EnumValues: []string{"percent", "amount"}},
		{Path: "advance_value", Type: "string", Description: "Advance amount or percent", SourceSection: "Dodanie faktury zaliczkowej na podstawie zamówienia – % pełnej kwoty"},
		{Path: "position_name", Type: "string", Description: "Advance position name", SourceSection: "Dodanie faktury zaliczkowej na podstawie zamówienia – % pełnej kwoty"},
		{Path: "invoice_id", Type: "integer", Description: "Linked invoice ID used in corrections and receipt links", SourceSection: "Dodanie nowej faktury korygującej"},
		{Path: "from_invoice_id", Type: "integer", Description: "Source invoice or receipt ID", SourceSection: "Dodanie nowej faktury korygującej"},
		{Path: "invoice_ids[]", Type: "integer", Description: "Referenced advance invoice IDs", SourceSection: "Dodanie faktury końcowej na podstawie zamówienia i faktur zaliczkowych"},
		{Path: "correction_reason", Type: "string", Description: "Correction reason", SourceSection: "Dodanie nowej faktury korygującej"},
		{Path: "gov_corrected_invoice_number", Type: "string", Description: "KSeF number of the corrected invoice", SourceSection: "Korekty faktur"},
		{Path: "additional_params", Type: "string", Description: "Additional upstream mode selector such as for_receipt", SourceSection: "Dodawanie faktury do istniejącego paragonu"},
		{Path: "exclude_from_stock_level", Type: "boolean", Description: "Do not affect stock levels", SourceSection: "Połączenie istniejącej faktury i paragonu"},
		{Path: "use_prices_from_price_lists", Type: "boolean", Description: "Use product prices from a price list", SourceSection: "Zaczytanie cen produktów z cennika podczas wystawiania faktury"},
		{Path: "price_list_id", Type: "integer", Description: "Price list ID used during product pricing", SourceSection: "Zaczytanie cen produktów z cennika podczas wystawiania faktury"},
		{Path: "show_attachments", Type: "boolean", Description: "Expose attachments to the customer on the public invoice page", SourceSection: "Dodanie nowego załącznika do faktury"},
		{Path: "exempt_tax_kind", Type: "string", Description: "Legal basis for a VAT-exempt position", SourceSection: "Zmiany w API przy tworzeniu faktur"},
		{Path: "np_tax_kind", Type: "string", Description: "Reason code for non-taxable positions", SourceSection: "Zmiany w API przy tworzeniu faktur", EnumValues: []string{"export_service", "export_service_eu"}},
		{Path: "positions[]", Type: "array<object>", Description: "Invoice positions", SourceSection: "Dodanie nowej faktury"},
		{Path: "positions[].id", Type: "integer", Description: "Invoice position ID", SourceSection: "Aktualizacja pozycji na fakturze"},
		{Path: "positions[]._destroy", Type: "boolean", Description: "Delete the referenced position during update", SourceSection: "Usunięcie pozycji na fakturze"},
		{Path: "positions[].product_id", Type: "integer", Description: "Referenced product ID", SourceSection: "Dodanie nowej faktury - minimalna wersja"},
		{Path: "positions[].name", Type: "string", Description: "Position name", SourceSection: "Dodanie nowej faktury"},
		{Path: "positions[].quantity", Type: "number", Description: "Position quantity", SourceSection: "Dodanie nowej faktury"},
		{Path: "positions[].tax", Type: "string", Description: "Position tax rate", SourceSection: "Dodanie nowej faktury"},
		{Path: "positions[].total_price_gross", Type: "number", Description: "Gross line total", SourceSection: "Dodanie nowej faktury"},
		{Path: "positions[].total_price_net", Type: "number", Description: "Net line total", SourceSection: "Dodanie nowej faktury"},
		{Path: "positions[].price_gross", Type: "number", Description: "Gross unit price", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
		{Path: "positions[].price_net", Type: "number", Description: "Net unit price", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
		{Path: "positions[].code", Type: "string", Description: "Product code", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
		{Path: "positions[].kind", Type: "string", Description: "Correction position kind", SourceSection: "Dodanie nowej faktury korygującej"},
		{
			Path:           "positions[].correction_before_attributes",
			Type:           "object",
			Description:    "Values before the correction",
			SourceSection:  "Dodanie nowej faktury korygującej",
			SchemaOverride: correctionAttributeSchema("Values before the correction"),
		},
		{
			Path:           "positions[].correction_after_attributes",
			Type:           "object",
			Description:    "Values after the correction",
			SourceSection:  "Dodanie nowej faktury korygującej",
			SchemaOverride: correctionAttributeSchema("Values after the correction"),
		},
		{Path: "descriptions[]", Type: "array<object>", Description: "Structured invoice notes", SourceSection: "Uwagi na fakturze (descriptions)"},
		{Path: "descriptions[].id", Type: "integer", Description: "Structured note ID", SourceSection: "Uwagi na fakturze (descriptions)"},
		{Path: "descriptions[]._destroy", Type: "boolean", Description: "Delete the referenced structured note during update", SourceSection: "Uwagi na fakturze (descriptions)"},
		{Path: "descriptions[].kind", Type: "string", Description: "Structured note heading", SourceSection: "Uwagi na fakturze (descriptions)"},
		{Path: "descriptions[].content", Type: "string", Description: "Structured note content", SourceSection: "Uwagi na fakturze (descriptions)"},
		{Path: "descriptions[].position_index", Type: "integer", Description: "Associated position index", SourceSection: "Uwagi na fakturze (descriptions)"},
		{Path: "descriptions[].row_number", Type: "integer", Description: "Display row number", SourceSection: "Uwagi na fakturze (descriptions)"},
		{Path: "settlement_positions[]", Type: "array<object>", Description: "Settlement adjustments applied to the invoice", SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
		{Path: "settlement_positions[].id", Type: "integer", Description: "Settlement position ID", SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
		{Path: "settlement_positions[]._destroy", Type: "boolean", Description: "Delete the referenced settlement position during update", SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
		{Path: "settlement_positions[].kind", Type: "string", Description: "Settlement kind: charge or deduction", SourceSection: "Rozliczenia na fakturze (settlement_positions)", EnumValues: []string{"charge", "deduction"}},
		{Path: "settlement_positions[].amount", Type: "number", Description: "Settlement amount", SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
		{Path: "settlement_positions[].reason", Type: "string", Description: "Settlement reason", SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
		{Path: "recipients[]", Type: "array<object>", Description: "Additional recipients", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "issuers[]", Type: "array<object>", Description: "Additional issuers", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].id", Type: "integer", Description: "Recipient ID", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[]._destroy", Type: "boolean", Description: "Delete the referenced recipient during update", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "issuers[].id", Type: "integer", Description: "Issuer ID", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "issuers[]._destroy", Type: "boolean", Description: "Delete the referenced issuer during update", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].name", Type: "string", Description: "Recipient name", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].first_name", Type: "string", Description: "Recipient first name", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].last_name", Type: "string", Description: "Recipient last name", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].tax_no", Type: "string", Description: "Recipient tax number", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].tax_no_kind", Type: "string", Description: "Recipient tax identifier kind", SourceSection: "Rodzaj identyfikatora podatkowego (tax_no_kind)", EnumValues: []string{"", "nip_ue", "other", "empty", "nip_with_id"}},
		{Path: "recipients[].company", Type: "boolean", Description: "Recipient company flag", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].country", Type: "string", Description: "Recipient country", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].city", Type: "string", Description: "Recipient city", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].post_code", Type: "string", Description: "Recipient postal code", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].street", Type: "string", Description: "Recipient street", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].phone", Type: "string", Description: "Recipient phone", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].email", Type: "string", Description: "Recipient email", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].note", Type: "string", Description: "Recipient note", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "recipients[].role", Type: "string", Description: "Recipient role", SourceSection: "Odbiorca faktury (Recipient)", EnumValues: []string{"Odbiorca", "Dodatkowy nabywca", "Dokonujący płatności", "JST – odbiorca", "Członek GV – odbiorca", "Pracownik", "Rola inna"}},
		{Path: "recipients[].role_description", Type: "string", Description: "Recipient custom role description", SourceSection: "Odbiorca faktury (Recipient)"},
		{Path: "recipients[].participation", Type: "number", Description: "Recipient participation percentage", SourceSection: "Odbiorca faktury (Recipient)"},
		{Path: "issuers[].name", Type: "string", Description: "Issuer name", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "issuers[].tax_no", Type: "string", Description: "Issuer tax number", SourceSection: "Wystawca faktury (Issuer)"},
		{Path: "issuers[].tax_no_kind", Type: "string", Description: "Issuer tax identifier kind", SourceSection: "Rodzaj identyfikatora podatkowego (tax_no_kind)", EnumValues: []string{"", "nip_ue", "other", "empty", "nip_with_id"}},
		{Path: "issuers[].company", Type: "boolean", Description: "Issuer company flag", SourceSection: "Wystawca faktury (Issuer)"},
		{Path: "issuers[].country", Type: "string", Description: "Issuer country", SourceSection: "Wystawca faktury (Issuer)"},
		{Path: "issuers[].email", Type: "string", Description: "Issuer email", SourceSection: "Odbiorcy/Wystawcy na fakturze"},
		{Path: "issuers[].role", Type: "string", Description: "Issuer role", SourceSection: "Wystawca faktury (Issuer)", EnumValues: []string{"Wystawca faktury", "Faktor", "Podmiot pierwotny", "JST – wystawca", "Członek GV – wystawca", "Rola inna"}},
		{Path: "issuers[].role_description", Type: "string", Description: "Issuer custom role description", SourceSection: "Wystawca faktury (Issuer)"},
	}
}

func correctionAttributeSchema(description string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": true,
		"description":          description,
		"properties": map[string]any{
			"name":              map[string]any{"type": "string"},
			"quantity":          map[string]any{"type": "number"},
			"total_price_gross": map[string]any{"type": "number"},
			"tax":               map[string]any{"type": "string"},
			"kind":              map[string]any{"type": "string"},
		},
	}
}

func recurringListOutputSpec() *OutputSpec {
	return recurringBaseOutputSpec("array", []string{"recurring list", "recurring create", "recurring update"})
}

func recurringGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"recurring list"}
	}
	return recurringBaseOutputSpec("object", commands)
}

func recurringRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "recurring",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax:  "dot_bracket",
		KnownFields: recurringRequestFields(),
		Notes: []string{
			"the CLI accepts the inner recurring object, then wraps it in the upstream {\"recurring\": ...} envelope",
			"known_fields is curated from the upstream README and is not exhaustive",
		},
	}
}

func recurringBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "name", "invoice_id", "every", "next_invoice_date", "send_email"},
		Notes: []string{
			"known_fields is curated from the upstream README and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
		},
		KnownFields: recurringKnownOutputFields(commands),
	}
}

func recurringKnownOutputFields(commands []string) []OutputFieldSpec {
	return []OutputFieldSpec{
		{Path: "id", Type: "integer", Description: "Recurring definition ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie listy definicji faktur cyklicznych"},
		{Path: "name", Type: "string", Description: "Recurring definition name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodanie definicji faktury cyklicznej"},
		{Path: "invoice_id", Type: "integer", Description: "Template invoice ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodanie definicji faktury cyklicznej"},
		{Path: "start_date", Type: "string", Description: "Recurring start date", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie definicji faktury cyklicznej"},
		{Path: "every", Type: "string", Description: "Recurring schedule expression", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodanie definicji faktury cyklicznej"},
		{Path: "issue_working_day_only", Type: "boolean", Description: "Only issue on working days", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie definicji faktury cyklicznej"},
		{Path: "send_email", Type: "boolean", Description: "Send invoice email automatically", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie definicji faktury cyklicznej"},
		{Path: "buyer_email", Type: "string", Description: "Email recipients used for auto-send", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie definicji faktury cyklicznej"},
		{Path: "end_date", Type: "string", Description: "Recurring end date", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie definicji faktury cyklicznej"},
		{Path: "next_invoice_date", Type: "string", Description: "Next invoice issue date", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Aktualizacja definicji faktury cyklicznej (zmiana daty wystawienia następnej faktury)"},
	}
}

func recurringRequestFields() []RequestFieldSpec {
	outputFields := recurringKnownOutputFields([]string{"recurring create", "recurring update"})
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
