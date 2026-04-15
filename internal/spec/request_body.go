package spec

import "github.com/sixers/fakturownia-cli/internal/output"

type RequestFieldSpec struct {
	Path           string         `json:"path"`
	Type           string         `json:"type"`
	Description    string         `json:"description"`
	Required       bool           `json:"required,omitempty"`
	EnumValues     []string       `json:"enum_values,omitempty"`
	SourceSection  string         `json:"source_section,omitempty"`
	SchemaOverride map[string]any `json:"-"`
}

type RequestBodySpec struct {
	InputFlag    string             `json:"input_flag"`
	InputModes   []string           `json:"input_modes"`
	WrapperKey   string             `json:"wrapper_key,omitempty"`
	OpenEnded    bool               `json:"open_ended"`
	CatalogBasis *CatalogBasis      `json:"catalog_basis,omitempty"`
	PathSyntax   string             `json:"path_syntax,omitempty"`
	KnownFields  []RequestFieldSpec `json:"known_fields,omitempty"`
	Notes        []string           `json:"notes,omitempty"`
}

func cloneRequestBodySpec(spec *RequestBodySpec) *RequestBodySpec {
	if spec == nil {
		return nil
	}
	return &RequestBodySpec{
		InputFlag:    spec.InputFlag,
		InputModes:   append([]string{}, spec.InputModes...),
		WrapperKey:   spec.WrapperKey,
		OpenEnded:    spec.OpenEnded,
		CatalogBasis: cloneCatalogBasis(spec.CatalogBasis),
		PathSyntax:   spec.PathSyntax,
		KnownFields:  cloneRequestFieldSpecs(spec.KnownFields),
		Notes:        append([]string{}, spec.Notes...),
	}
}

func cloneRequestFieldSpecs(fields []RequestFieldSpec) []RequestFieldSpec {
	cloned := make([]RequestFieldSpec, 0, len(fields))
	for _, field := range fields {
		field.EnumValues = append([]string{}, field.EnumValues...)
		field.SchemaOverride = cloneSchemaMap(field.SchemaOverride)
		cloned = append(cloned, field)
	}
	return cloned
}

func buildRequestBodySchema(spec *RequestBodySpec) (map[string]any, error) {
	if spec == nil {
		return nil, nil
	}
	objectSchema := newOpenObjectSchema()
	for _, field := range spec.KnownFields {
		path, err := output.ParsePath(field.Path)
		if err != nil {
			return nil, err
		}
		applyRequestFieldSchema(objectSchema, path.Segments, field)
	}
	return objectSchema, nil
}

func applyRequestFieldSchema(root map[string]any, segments []output.PathSegment, field RequestFieldSpec) {
	if len(segments) == 1 && field.SchemaOverride != nil {
		setExplicitRequestPropertySchema(root, segments[0], field)
		return
	}
	outputField := OutputFieldSpec{
		Path:        field.Path,
		Type:        field.Type,
		Description: field.Description,
		EnumValues:  append([]string{}, field.EnumValues...),
	}
	applyFieldSchema(root, segments, outputField)
}

func setExplicitRequestPropertySchema(root map[string]any, segment output.PathSegment, field RequestFieldSpec) {
	properties := ensureProperties(root)
	schema := cloneSchemaMap(field.SchemaOverride)
	if schema == nil {
		schema = newOpenObjectSchema()
	}
	if field.Description != "" {
		schema["description"] = field.Description
	}
	if field.Required {
		required := ensureRequired(root)
		if !containsString(required, segment.Name) {
			root["required"] = append(required, segment.Name)
		}
	}
	properties[segment.Name] = schema
}

func cloneCatalogBasis(value *CatalogBasis) *CatalogBasis {
	if value == nil {
		return nil
	}
	return &CatalogBasis{
		Source: value.Source,
		URL:    value.URL,
	}
}

func ensureRequired(schema map[string]any) []string {
	if existing, ok := schema["required"].([]string); ok {
		return append([]string{}, existing...)
	}
	if existing, ok := schema["required"].([]any); ok {
		out := make([]string, 0, len(existing))
		for _, item := range existing {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
