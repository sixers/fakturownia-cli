package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sixers/fakturownia-cli/internal/output"
)

func ParseInput(raw string, stdin io.Reader) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, output.Usage("missing_input", "client input is required", "pass --input -|@file.json|'{...}'")
	}

	data, err := readInput(trimmed, stdin)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, output.Usage("empty_input", "client input cannot be empty", "provide a JSON object via --input, @file, or stdin")
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
		return nil, output.Usage("invalid_input_shape", "--input must be a JSON object", "pass the inner client object, for example '{\"name\":\"Acme\"}'")
	}
	return object, nil
}

func readInput(raw string, stdin io.Reader) ([]byte, error) {
	switch {
	case raw == "-":
		if stdin == nil {
			return nil, output.Internal(nil, "stdin is not available")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, output.Internal(err, "read client input from stdin")
		}
		return data, nil
	case strings.HasPrefix(raw, "@"):
		path := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
		if path == "" {
			return nil, output.Usage("invalid_input_path", "--input @file requires a file path", "pass --input @/absolute/or/relative/path.json")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, output.Internal(err, "read client input file")
		}
		return data, nil
	default:
		return []byte(raw), nil
	}
}
