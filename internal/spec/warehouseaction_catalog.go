package spec

type warehouseActionFieldDefinition struct {
	Name          string
	Type          string
	Description   string
	SourceSection string
}

func warehouseActionListOutputSpec() *OutputSpec {
	commands := []string{"warehouse-action list"}
	fields := []OutputFieldSpec{
		{Path: "id", Type: "integer", Description: "Warehouse action ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Lista wszystkich akcji magazynowych"},
		{Path: "kind", Type: "string", Description: "Warehouse action kind", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Lista wszystkich akcji magazynowych"},
		{Path: "warehouse_id", Type: "integer", Description: "Warehouse ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Lista wszystkich akcji magazynowych"},
		{Path: "warehouse_document_id", Type: "integer", Description: "Linked warehouse document ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Lista wszystkich akcji magazynowych"},
	}
	fields = append(fields, warehouseActionLeafOutputFields("", commands)...)

	return &OutputSpec{
		Shape:      "array",
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "kind", "product_id", "quantity", "warehouse_id", "warehouse_document_id"},
		Notes: []string{
			"known_fields is curated from the upstream README and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
			"warehouse-action list exposes only explicit first-class filters for the README-documented query params in v1",
		},
		KnownFields: fields,
	}
}

func warehouseActionLeafDefinitions() []warehouseActionFieldDefinition {
	return []warehouseActionFieldDefinition{
		{Name: "product_id", Type: "integer", Description: "Referenced product ID", SourceSection: "Dodawanie dokumentu MM"},
		{Name: "product_name", Type: "string", Description: "Product name", SourceSection: "Dodawanie dokumentu PZ"},
		{Name: "purchase_tax", Type: "string", Description: "Purchase tax rate", SourceSection: "Dodawanie dokumentu PZ"},
		{Name: "purchase_price_net", Type: "number", Description: "Net purchase price", SourceSection: "Dodawanie dokumentu PZ"},
		{Name: "tax", Type: "string", Description: "Sales tax rate", SourceSection: "Dodawanie dokumentu WZ"},
		{Name: "price_net", Type: "number", Description: "Net price", SourceSection: "Dodawanie dokumentu WZ"},
		{Name: "quantity", Type: "number", Description: "Quantity", SourceSection: "Dodawanie dokumentu MM"},
		{Name: "warehouse2_id", Type: "integer", Description: "Target warehouse ID for transfers", SourceSection: "Dodawanie dokumentu MM"},
	}
}

func warehouseActionLeafOutputFields(prefix string, commands []string) []OutputFieldSpec {
	definitions := warehouseActionLeafDefinitions()
	fields := make([]OutputFieldSpec, 0, len(definitions))
	for _, def := range definitions {
		path := def.Name
		if prefix != "" {
			path = prefix + "." + def.Name
		}
		fields = append(fields, OutputFieldSpec{
			Path:          path,
			Type:          def.Type,
			Description:   def.Description,
			Projectable:   true,
			Selectable:    true,
			Commands:      commands,
			Presence:      "conditional",
			SourceSection: def.SourceSection,
		})
	}
	return fields
}

func warehouseActionLeafRequestFields(prefix string) []RequestFieldSpec {
	definitions := warehouseActionLeafDefinitions()
	fields := make([]RequestFieldSpec, 0, len(definitions))
	for _, def := range definitions {
		path := def.Name
		if prefix != "" {
			path = prefix + "." + def.Name
		}
		fields = append(fields, RequestFieldSpec{
			Path:          path,
			Type:          def.Type,
			Description:   def.Description,
			SourceSection: def.SourceSection,
		})
	}
	return fields
}
