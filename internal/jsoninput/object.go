package jsoninput

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sixers/fakturownia-cli/internal/output"
)

func ParseObject(raw string, stdin io.Reader, noun string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, output.Usage("missing_input", fmt.Sprintf("%s input is required", noun), "pass --input -|@file.json|'{...}'")
	}

	data, err := readInput(trimmed, stdin, noun)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, output.Usage("empty_input", fmt.Sprintf("%s input cannot be empty", noun), "provide a JSON object via --input, @file, or stdin")
	}

	var value any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil, output.Usage("invalid_input_json", fmt.Sprintf("could not parse --input JSON: %v", err), "provide a valid JSON object")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, output.Usage("invalid_input_json", "--input must contain a single JSON value", "provide one JSON object")
	}

	object, ok := value.(map[string]any)
	if !ok {
		return nil, output.Usage("invalid_input_shape", "--input must be a JSON object", fmt.Sprintf("pass the inner %s object, for example '{\"name\":\"Acme\"}'", noun))
	}
	return object, nil
}

func readInput(raw string, stdin io.Reader, noun string) ([]byte, error) {
	switch {
	case raw == "-":
		if stdin == nil {
			return nil, output.Internal(nil, "stdin is not available")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, output.Internal(err, fmt.Sprintf("read %s input from stdin", noun))
		}
		return data, nil
	case strings.HasPrefix(raw, "@"):
		path := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
		if path == "" {
			return nil, output.Usage("invalid_input_path", "--input @file requires a file path", "pass --input @/absolute/or/relative/path.json")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, output.Internal(err, fmt.Sprintf("read %s input file", noun))
		}
		return data, nil
	default:
		return []byte(raw), nil
	}
}
