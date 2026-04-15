package jsoninput

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseObjectSources(t *testing.T) {
	t.Parallel()

	inline, err := ParseObject(`{"name":"Acme"}`, nil, "product")
	if err != nil || inline["name"] != "Acme" {
		t.Fatalf("ParseObject(inline) = %#v, %v", inline, err)
	}

	stdin, err := ParseObject("-", strings.NewReader(`{"name":"stdin"}`), "product")
	if err != nil || stdin["name"] != "stdin" {
		t.Fatalf("ParseObject(stdin) = %#v, %v", stdin, err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "input.json")
	if err := os.WriteFile(path, []byte(`{"name":"file"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	fromFile, err := ParseObject("@"+path, nil, "product")
	if err != nil || fromFile["name"] != "file" {
		t.Fatalf("ParseObject(file) = %#v, %v", fromFile, err)
	}
}

func TestParseObjectRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		raw   string
		stdin io.Reader
	}{
		{name: "empty", raw: ""},
		{name: "array", raw: `[]`},
		{name: "invalid", raw: `{`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseObject(tc.raw, tc.stdin, "product"); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
