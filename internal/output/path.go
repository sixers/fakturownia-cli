package output

import (
	"fmt"
	"regexp"
	"strings"
)

var fieldNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

type PathSegment struct {
	Name  string
	Array bool
}

type FieldPath struct {
	Raw      string
	Segments []PathSegment
}

func ParsePaths(rawPaths []string) ([]FieldPath, error) {
	paths := make([]FieldPath, 0, len(rawPaths))
	for _, raw := range rawPaths {
		path, err := ParsePath(raw)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func ParsePath(raw string) (FieldPath, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return FieldPath{}, fmt.Errorf("field path cannot be empty")
	}
	if strings.HasPrefix(trimmed, "$") {
		return FieldPath{}, fmt.Errorf("field path %q must not start with $", raw)
	}

	parts := strings.Split(trimmed, ".")
	segments := make([]PathSegment, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			return FieldPath{}, fmt.Errorf("field path %q contains an empty segment", raw)
		}
		if strings.Contains(part, "*") {
			return FieldPath{}, fmt.Errorf("field path %q contains unsupported wildcard syntax", raw)
		}

		array := strings.HasSuffix(part, "[]")
		name := strings.TrimSuffix(part, "[]")
		if name == "" {
			return FieldPath{}, fmt.Errorf("field path %q contains an empty segment", raw)
		}
		if strings.ContainsAny(name, "[]") {
			return FieldPath{}, fmt.Errorf("field path %q uses unsupported bracket syntax", raw)
		}
		if !fieldNamePattern.MatchString(name) {
			return FieldPath{}, fmt.Errorf("field path %q contains invalid segment %q", raw, name)
		}

		segments = append(segments, PathSegment{Name: name, Array: array})
	}

	return FieldPath{Raw: trimmed, Segments: segments}, nil
}

func ExtractPathValue(data any, raw string) (any, bool, error) {
	path, err := ParsePath(raw)
	if err != nil {
		return nil, false, err
	}

	generic, err := toGeneric(data)
	if err != nil {
		return nil, false, err
	}
	value, ok := extractPathSegments(generic, path.Segments)
	return value, ok, nil
}

func ProjectData(data any, fields []string) (any, error) {
	if len(fields) == 0 {
		return data, nil
	}

	paths, err := ParsePaths(fields)
	if err != nil {
		return nil, err
	}

	generic, err := toGeneric(data)
	if err != nil {
		return nil, err
	}

	switch typed := generic.(type) {
	case map[string]any:
		return projectMapWithPaths(typed, paths), nil
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			record, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("field projection only supports objects and lists of objects")
			}
			out = append(out, projectMapWithPaths(record, paths))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("field projection only supports objects and lists of objects")
	}
}

func projectMapWithPaths(record map[string]any, paths []FieldPath) map[string]any {
	projected := make(map[string]any)
	for _, path := range paths {
		partial, ok := projectObjectPath(record, path.Segments)
		if !ok {
			continue
		}
		mergeProjectedMaps(projected, partial)
	}
	return projected
}

func projectObjectPath(record map[string]any, segments []PathSegment) (map[string]any, bool) {
	if len(segments) == 0 {
		return nil, false
	}

	segment := segments[0]
	value, ok := record[segment.Name]
	if !ok {
		return nil, false
	}

	if len(segments) == 1 {
		if segment.Array {
			items, ok := value.([]any)
			if !ok {
				return nil, false
			}
			return map[string]any{segment.Name: cloneGeneric(items)}, true
		}
		return map[string]any{segment.Name: cloneGeneric(value)}, true
	}

	if segment.Array {
		items, ok := value.([]any)
		if !ok {
			return nil, false
		}

		projectedItems := make([]any, len(items))
		anyProjected := false
		for idx, item := range items {
			child, ok := item.(map[string]any)
			if !ok {
				projectedItems[idx] = map[string]any{}
				continue
			}
			partial, ok := projectObjectPath(child, segments[1:])
			if !ok {
				projectedItems[idx] = map[string]any{}
				continue
			}
			projectedItems[idx] = partial
			anyProjected = true
		}
		if !anyProjected {
			return nil, false
		}
		return map[string]any{segment.Name: projectedItems}, true
	}

	child, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	partial, ok := projectObjectPath(child, segments[1:])
	if !ok {
		return nil, false
	}
	return map[string]any{segment.Name: partial}, true
}

func mergeProjectedMaps(dst, src map[string]any) {
	for key, value := range src {
		existing, ok := dst[key]
		if !ok {
			dst[key] = cloneGeneric(value)
			continue
		}

		switch typed := value.(type) {
		case map[string]any:
			existingMap, ok := existing.(map[string]any)
			if !ok {
				dst[key] = cloneGeneric(value)
				continue
			}
			mergeProjectedMaps(existingMap, typed)
		case []any:
			existingArray, ok := existing.([]any)
			if !ok {
				dst[key] = cloneGeneric(value)
				continue
			}
			dst[key] = mergeProjectedArrays(existingArray, typed)
		default:
			dst[key] = typed
		}
	}
}

func mergeProjectedArrays(dst, src []any) []any {
	if len(dst) < len(src) {
		expanded := make([]any, len(src))
		copy(expanded, dst)
		dst = expanded
	}

	for idx, value := range src {
		if value == nil {
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			existingMap, ok := dst[idx].(map[string]any)
			if !ok {
				dst[idx] = cloneGeneric(value)
				continue
			}
			mergeProjectedMaps(existingMap, typed)
		case []any:
			existingArray, ok := dst[idx].([]any)
			if !ok {
				dst[idx] = cloneGeneric(value)
				continue
			}
			dst[idx] = mergeProjectedArrays(existingArray, typed)
		default:
			dst[idx] = typed
		}
	}

	return dst
}

func extractPathValue(current any, path FieldPath) any {
	value, ok := extractPathSegments(current, path.Segments)
	if !ok {
		return nil
	}
	return value
}

func extractPathSegments(current any, segments []PathSegment) (any, bool) {
	if len(segments) == 0 {
		return current, true
	}

	record, ok := current.(map[string]any)
	if !ok {
		return nil, false
	}

	segment := segments[0]
	value, ok := record[segment.Name]
	if !ok {
		return nil, false
	}

	if segment.Array {
		items, ok := value.([]any)
		if !ok {
			return nil, false
		}
		if len(segments) == 1 {
			return cloneGeneric(items), true
		}

		values := make([]any, 0, len(items))
		for _, item := range items {
			child, ok := extractPathSegments(item, segments[1:])
			if !ok {
				continue
			}
			if nested, ok := child.([]any); ok {
				values = append(values, nested...)
				continue
			}
			values = append(values, child)
		}
		if len(values) == 0 {
			return nil, false
		}
		return values, true
	}

	if len(segments) == 1 {
		return cloneGeneric(value), true
	}
	return extractPathSegments(value, segments[1:])
}

func cloneGeneric(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = cloneGeneric(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for idx, child := range typed {
			out[idx] = cloneGeneric(child)
		}
		return out
	default:
		return typed
	}
}
