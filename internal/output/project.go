package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func ProjectData(data any, fields []string) (any, error) {
	if len(fields) == 0 {
		return data, nil
	}

	generic, err := toGeneric(data)
	if err != nil {
		return nil, err
	}

	switch typed := generic.(type) {
	case map[string]any:
		return projectMap(typed, fields), nil
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			if record, ok := item.(map[string]any); ok {
				out = append(out, projectMap(record, fields))
				continue
			}
			return nil, fmt.Errorf("field projection only supports objects and lists of objects")
		}
		return out, nil
	default:
		return nil, fmt.Errorf("field projection only supports objects and lists of objects")
	}
}

func ToPrettyJSON(data any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func toGeneric(data any) (any, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func projectMap(record map[string]any, fields []string) map[string]any {
	projected := make(map[string]any, len(fields))
	for _, field := range fields {
		if value, ok := record[strings.TrimSpace(field)]; ok {
			projected[strings.TrimSpace(field)] = value
		}
	}
	return projected
}
