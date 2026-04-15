package spec

import "github.com/sixers/fakturownia-cli/internal/output"

type RequestFieldSpec struct {
	Path          string   `json:"path"`
	Type          string   `json:"type"`
	Description   string   `json:"description"`
	Required      bool     `json:"required,omitempty"`
	EnumValues    []string `json:"enum_values,omitempty"`
	SourceSection string   `json:"source_section,omitempty"`
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
		KnownFields:  append([]RequestFieldSpec{}, spec.KnownFields...),
		Notes:        append([]string{}, spec.Notes...),
	}
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
	outputField := OutputFieldSpec{
		Path:        field.Path,
		Type:        field.Type,
		Description: field.Description,
		EnumValues:  append([]string{}, field.EnumValues...),
	}
	applyFieldSchema(root, segments, outputField)
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
