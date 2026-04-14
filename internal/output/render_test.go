package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestProjectData(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"id": 1, "number": "FV/1", "status": "issued"},
		{"id": 2, "number": "FV/2", "status": "paid"},
	}

	projected, err := ProjectData(data, []string{"id", "number"})
	if err != nil {
		t.Fatalf("ProjectData() error = %v", err)
	}

	got := projected.([]any)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
}

func TestRenderSuccessJSONProjection(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := RenderSuccess(&stdout, Options{
		Format: "json",
		Fields: []string{"id", "number"},
	}, Result{
		Data: map[string]any{
			"id":     1,
			"number": "FV/1",
			"status": "issued",
		},
		Meta: Meta{Command: "invoice get"},
	})
	if err != nil {
		t.Fatalf("RenderSuccess() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, `"id": 1`) || !strings.Contains(out, `"number": "FV/1"`) {
		t.Fatalf("expected projected fields in output, got %s", out)
	}
	if strings.Contains(out, `"status": "issued"`) {
		t.Fatalf("did not expect omitted field in output: %s", out)
	}
}

func TestRenderSuccessQuiet(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := RenderSuccess(&stdout, Options{
		Format: "human",
		Quiet:  true,
		Fields: []string{"number"},
	}, Result{
		Data: []map[string]any{
			{"number": "FV/1"},
			{"number": "FV/2"},
		},
		Meta: Meta{Command: "invoice list"},
	})
	if err != nil {
		t.Fatalf("RenderSuccess() error = %v", err)
	}

	if got := stdout.String(); got != "FV/1\nFV/2\n" {
		t.Fatalf("unexpected quiet output: %q", got)
	}
}
