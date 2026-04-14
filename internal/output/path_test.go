package output

import (
	"reflect"
	"testing"
)

func TestParsePath(t *testing.T) {
	t.Parallel()

	valid := []string{
		"number",
		"status",
		"positions[].name",
		"recipients[].address.city",
	}
	for _, raw := range valid {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			if _, err := ParsePath(raw); err != nil {
				t.Fatalf("ParsePath() error = %v", err)
			}
		})
	}

	invalid := []string{
		"",
		"positions[0].name",
		"$.number",
		"positions.*.name",
		"positions..name",
		".number",
	}
	for _, raw := range invalid {
		raw := raw
		t.Run("invalid/"+raw, func(t *testing.T) {
			if _, err := ParsePath(raw); err == nil {
				t.Fatalf("expected ParsePath(%q) to fail", raw)
			}
		})
	}
}

func TestProjectDataNestedFields(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"id":     1,
		"number": "FV/1",
		"buyer": map[string]any{
			"name":   "Acme",
			"tax_no": "PL123",
		},
		"positions": []any{
			map[string]any{"name": "A", "tax": "23", "quantity": 1},
			map[string]any{"name": "B", "tax": "8", "quantity": 2},
		},
	}

	projected, err := ProjectData(data, []string{"number", "buyer.name", "positions[].name", "positions[].tax"})
	if err != nil {
		t.Fatalf("ProjectData() error = %v", err)
	}

	want := map[string]any{
		"number": "FV/1",
		"buyer": map[string]any{
			"name": "Acme",
		},
		"positions": []any{
			map[string]any{"name": "A", "tax": "23"},
			map[string]any{"name": "B", "tax": "8"},
		},
	}
	if !reflect.DeepEqual(projected, want) {
		t.Fatalf("unexpected projection\nwant: %#v\ngot:  %#v", want, projected)
	}
}

func TestProjectDataListNestedFields(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{
			"number": "FV/1",
			"positions": []any{
				map[string]any{"name": "A", "tax": "23"},
			},
		},
		{
			"number": "FV/2",
			"positions": []any{
				map[string]any{"name": "B", "tax": "8"},
			},
		},
	}

	projected, err := ProjectData(data, []string{"number", "positions[].name"})
	if err != nil {
		t.Fatalf("ProjectData() error = %v", err)
	}

	got := projected.([]any)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}

	first := got[0].(map[string]any)
	if first["number"] != "FV/1" {
		t.Fatalf("expected first row number, got %#v", first)
	}
	positions := first["positions"].([]any)
	if positions[0].(map[string]any)["name"] != "A" {
		t.Fatalf("expected nested projected position name, got %#v", positions[0])
	}
}

func TestExtractPathValue(t *testing.T) {
	t.Parallel()

	record := map[string]any{
		"positions": []any{
			map[string]any{"name": "A"},
			map[string]any{"name": "B"},
		},
	}

	value, ok, err := ExtractPathValue(record, "positions[].name")
	if err != nil {
		t.Fatalf("ExtractPathValue() error = %v", err)
	}
	if !ok {
		t.Fatal("expected path lookup to succeed")
	}

	got := value.([]any)
	want := []any{"A", "B"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected extracted values\nwant: %#v\ngot:  %#v", want, got)
	}
}
