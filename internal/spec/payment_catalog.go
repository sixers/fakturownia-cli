package spec

func paymentListOutputSpec() *OutputSpec {
	return paymentBaseOutputSpec("array", []string{"payment list"})
}

func paymentGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"payment get", "payment create", "payment update"}
	}
	return paymentBaseOutputSpec("object", commands)
}

func paymentRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "banking_payment",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax: "dot_bracket",
		KnownFields: []RequestFieldSpec{
			{Path: "name", Type: "string", Description: "Payment name", SourceSection: "Dodawanie nowej płatności"},
			{Path: "price", Type: "number", Description: "Payment amount", SourceSection: "Dodawanie nowej płatności"},
			{Path: "invoice_id", Type: "integer|null", Description: "Single linked invoice ID", SourceSection: "Dodawanie nowej płatności"},
			{Path: "invoice_ids[]", Type: "integer", Description: "Ordered linked invoice IDs", SourceSection: "Dodanie nowej płatności powiązanej z istniejącymi fakturami"},
			{Path: "paid", Type: "boolean", Description: "Paid flag", SourceSection: "Dodawanie nowej płatności"},
			{Path: "kind", Type: "string", Description: "Payment kind", SourceSection: "Dodawanie nowej płatności"},
		},
		Notes: []string{
			"the CLI accepts the inner banking payment object, then wraps it in the upstream {\"banking_payment\": ...} envelope",
			"known_fields is curated from the upstream README and is not exhaustive",
			"invoice_ids pays invoices in the order provided by the upstream API",
		},
	}
}

func paymentBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "name", "price", "paid", "kind"},
		Notes: []string{
			"known_fields is curated from the upstream README and verified endpoint behavior and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
			"payment get uses the verified singular /banking/payment/:id.json endpoint, while list, update, and delete use /banking/payments",
		},
		KnownFields: []OutputFieldSpec{
			{Path: "id", Type: "integer", Description: "Payment ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie wybranej płatności po ID"},
			{Path: "name", Type: "string", Description: "Payment name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodawanie nowej płatności"},
			{Path: "price", Type: "number", Description: "Payment amount", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodawanie nowej płatności"},
			{Path: "paid", Type: "boolean", Description: "Paid flag", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodawanie nowej płatności"},
			{Path: "kind", Type: "string", Description: "Payment kind", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodawanie nowej płatności"},
			{Path: "invoice_id", Type: "integer|null", Description: "Single linked invoice ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dodawanie nowej płatności"},
			{Path: "invoices[]", Type: "array<object>", Description: "Invoices included when requested through the upstream include=invoices query", Projectable: true, Selectable: false, Commands: []string{"payment list"}, Presence: "conditional", Requires: []string{"payment list --include invoices"}, SourceSection: "Pobranie płatności wraz z danymi przypiętych faktur"},
		},
	}
}
