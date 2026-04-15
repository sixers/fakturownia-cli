package spec

func productListOutputSpec() *OutputSpec {
	return productBaseOutputSpec("array", []string{"product list", "product get"})
}

func productGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"product get"}
	}
	return productBaseOutputSpec("object", commands)
}

func productCreateRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "product",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax:  "dot_bracket",
		KnownFields: productRequestFields(),
		Notes: []string{
			"the CLI accepts the inner product object, then wraps it in the upstream {\"product\": ...} envelope",
			"known_fields is curated from the upstream README and is not exhaustive",
			"package_products_details is an open object keyed by arbitrary indexes, where each value contains at least id and quantity",
		},
	}
}

func productUpdateRequestBodySpec() *RequestBodySpec {
	spec := productCreateRequestBodySpec()
	spec.Notes = append(append([]string{}, spec.Notes...),
		`the upstream README states that price_net is computed from price_gross and tax during product updates; the CLI documents this but does not rewrite or reject the field client-side`,
	)
	return spec
}

func productBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "name", "code", "price_gross", "tax", "stock_level"},
		Notes: []string{
			"known_fields is curated from the upstream README and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
		},
		KnownFields: productKnownOutputFields(commands),
	}
}

func productKnownOutputFields(commands []string) []OutputFieldSpec {
	return []OutputFieldSpec{
		{Path: "id", Type: "integer", Description: "Product ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie wybranego produktu po ID"},
		{Path: "name", Type: "string", Description: "Product name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pola produktu"},
		{Path: "code", Type: "string", Description: "Product code", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pola produktu"},
		{Path: "ean_code", Type: "string", Description: "EAN code", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "description", Type: "string", Description: "Product description", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "price_net", Type: "number", Description: "Net sales price", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola produktu"},
		{Path: "tax", Type: "string", Description: "Tax rate or tax status", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pola produktu", EnumValues: []string{"np", "zw", "disabled"}},
		{Path: "price_gross", Type: "number", Description: "Gross sales price", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pola produktu"},
		{Path: "currency", Type: "string", Description: "Currency", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola produktu"},
		{Path: "category_id", Type: "integer", Description: "Category ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "tag_list[]", Type: "string", Description: "Product tags", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "service", Type: "string", Description: "Service flag", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pola produktu", EnumValues: []string{"1", "0", "true", "false"}},
		{Path: "electronic_service", Type: "string", Description: "Electronic service flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu", EnumValues: []string{"1", "0"}},
		{Path: "gtu_codes[]", Type: "string", Description: "GTU codes", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "limited", Type: "string", Description: "Quantity-limited flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu", EnumValues: []string{"1", "0"}},
		{Path: "stock_level", Type: "number", Description: "Available stock quantity", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "purchase_price_net", Type: "number", Description: "Net purchase price", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "purchase_tax", Type: "string", Description: "Purchase tax rate", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "purchase_price_gross", Type: "number", Description: "Gross purchase price", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "package", Type: "string", Description: "Package-product flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie produktu, który jest zestawem", EnumValues: []string{"1", "0", "true", "false"}},
		{Path: "quantity_unit", Type: "string", Description: "Quantity unit", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "quantity", Type: "number", Description: "Default sold quantity", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "additional_info", Type: "string", Description: "Additional classification info", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "supplier_code", Type: "string", Description: "Supplier code", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "accounting_id", Type: "string", Description: "Accounting code", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "disabled", Type: "string", Description: "Disabled flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu", EnumValues: []string{"1", "0"}},
		{Path: "use_moss", Type: "string", Description: "OSS/MOSS flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu", EnumValues: []string{"1", "0"}},
		{Path: "use_product_warehouses", Type: "string", Description: "Separate warehouse pricing flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu", EnumValues: []string{"1", "0"}},
		{Path: "size", Type: "string", Description: "Size", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "size_width", Type: "string", Description: "Width", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "size_height", Type: "string", Description: "Height", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "size_unit", Type: "string", Description: "Size unit", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu", EnumValues: []string{"m", "cm"}},
		{Path: "weight", Type: "string", Description: "Weight", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu"},
		{Path: "weight_unit", Type: "string", Description: "Weight unit", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pola produktu", EnumValues: []string{"kg", "g"}},
	}
}

func productRequestFields() []RequestFieldSpec {
	outputFields := productKnownOutputFields([]string{"product create", "product update"})
	fields := make([]RequestFieldSpec, 0, len(outputFields)+1)
	for _, field := range outputFields {
		fields = append(fields, RequestFieldSpec{
			Path:          field.Path,
			Type:          field.Type,
			Description:   field.Description,
			EnumValues:    append([]string{}, field.EnumValues...),
			SourceSection: field.SourceSection,
		})
	}
	fields = append(fields, RequestFieldSpec{
		Path:          "package_products_details",
		Type:          "object",
		Description:   "Open object keyed by arbitrary indexes, where each value contains package product id and quantity",
		SourceSection: "Dodanie produktu, który jest zestawem",
		SchemaOverride: map[string]any{
			"type":                 "object",
			"additionalProperties": packageProductDetailSchema(),
		},
	})
	return fields
}

func packageProductDetailSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": true,
		"required":             []string{"id", "quantity"},
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "integer",
				"description": "Referenced product ID",
			},
			"quantity": map[string]any{
				"type":        "number",
				"description": "Quantity of the referenced product in the package",
			},
		},
	}
}
