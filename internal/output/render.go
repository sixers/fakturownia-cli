package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Options struct {
	Format  string
	Raw     bool
	Quiet   bool
	Fields  []string
	Columns []string
}

type Result struct {
	Data           any
	RawBody        []byte
	Meta           Meta
	Warnings       []WarningDetail
	HumanRenderer  HumanRenderer
	DefaultColumns []string
}

type HumanInput struct {
	Data           any
	Fields         []string
	Columns        []string
	DefaultColumns []string
}

type HumanRenderer interface {
	Render(io.Writer, HumanInput) error
	QuietValues(HumanInput) ([]string, error)
}

type JSONRenderer struct{}

func (JSONRenderer) Render(w io.Writer, input HumanInput) error {
	data := input.Data
	if len(input.Fields) > 0 {
		projected, err := ProjectData(input.Data, input.Fields)
		if err != nil {
			return err
		}
		data = projected
	}
	out, err := ToPrettyJSON(data)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

func (JSONRenderer) QuietValues(input HumanInput) ([]string, error) {
	return quietValuesFromData(input.Data, input.Fields)
}

type TableRenderer struct{}

func (TableRenderer) Render(w io.Writer, input HumanInput) error {
	rows, err := toRows(input.Data)
	if err != nil {
		return err
	}
	columns := selectedColumns(input.Columns, input.DefaultColumns)
	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(columns, "\t"))
	for _, row := range rows {
		values := make([]string, 0, len(columns))
		for _, column := range columns {
			values = append(values, stringify(row[column]))
		}
		fmt.Fprintln(tw, strings.Join(values, "\t"))
	}
	return tw.Flush()
}

func (TableRenderer) QuietValues(input HumanInput) ([]string, error) {
	rows, err := toRows(input.Data)
	if err != nil {
		return nil, err
	}
	columns := selectedColumns(input.Columns, input.DefaultColumns)
	if len(columns) != 1 {
		return nil, fmt.Errorf("quiet mode requires exactly one column")
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, stringify(row[columns[0]]))
	}
	return lines, nil
}

type LinesRenderer struct {
	Lines func(any) ([]string, error)
}

func (r LinesRenderer) Render(w io.Writer, input HumanInput) error {
	lines, err := r.Lines(input.Data)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func (r LinesRenderer) QuietValues(input HumanInput) ([]string, error) {
	return r.Lines(input.Data)
}

func RenderSuccess(stdout io.Writer, opts Options, result Result) error {
	if opts.Raw {
		if _, err := stdout.Write(result.RawBody); err != nil {
			return err
		}
		return nil
	}

	if opts.Format == "json" {
		data := result.Data
		if len(opts.Fields) > 0 {
			projected, err := ProjectData(data, opts.Fields)
			if err != nil {
				return err
			}
			data = projected
		}
		return writeJSON(stdout, Envelope{
			SchemaVersion: SchemaVersion,
			Status:        "success",
			Data:          data,
			Errors:        []ErrorDetail{},
			Warnings:      result.Warnings,
			Meta:          result.Meta,
		})
	}

	if opts.Quiet {
		lines, err := quietValues(result, opts)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if _, err := fmt.Fprintln(stdout, line); err != nil {
				return err
			}
		}
		return nil
	}

	if result.HumanRenderer == nil {
		return JSONRenderer{}.Render(stdout, HumanInput{
			Data:           result.Data,
			Fields:         opts.Fields,
			Columns:        opts.Columns,
			DefaultColumns: result.DefaultColumns,
		})
	}

	return result.HumanRenderer.Render(stdout, HumanInput{
		Data:           result.Data,
		Fields:         opts.Fields,
		Columns:        opts.Columns,
		DefaultColumns: result.DefaultColumns,
	})
}

func RenderError(stdout, stderr io.Writer, opts Options, meta Meta, err *AppError) error {
	if err == nil {
		return nil
	}

	if opts.Raw && len(err.RawBody()) > 0 {
		_, writeErr := stdout.Write(err.RawBody())
		return writeErr
	}

	if opts.Format == "json" {
		return writeJSON(stdout, Envelope{
			SchemaVersion: SchemaVersion,
			Status:        "error",
			Data:          nil,
			Errors:        []ErrorDetail{err.Detail()},
			Warnings:      []WarningDetail{},
			Meta:          meta,
		})
	}

	_, writeErr := fmt.Fprintf(stderr, "%s (%s)\n", err.Detail().Message, err.Detail().Code)
	if writeErr != nil {
		return writeErr
	}
	if hint := err.Detail().Hint; hint != "" {
		_, writeErr = fmt.Fprintf(stderr, "hint: %s\n", hint)
	}
	return writeErr
}

func writeJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func quietValues(result Result, opts Options) ([]string, error) {
	if len(opts.Fields) > 0 {
		return quietValuesFromData(result.Data, opts.Fields)
	}
	if result.HumanRenderer == nil {
		return quietValuesFromData(result.Data, nil)
	}
	return result.HumanRenderer.QuietValues(HumanInput{
		Data:           result.Data,
		Fields:         opts.Fields,
		Columns:        opts.Columns,
		DefaultColumns: result.DefaultColumns,
	})
}

func quietValuesFromData(data any, fields []string) ([]string, error) {
	target := data
	if len(fields) > 0 {
		projected, err := ProjectData(data, fields)
		if err != nil {
			return nil, err
		}
		target = projected
	}

	generic, err := toGeneric(target)
	if err != nil {
		return nil, err
	}

	switch typed := generic.(type) {
	case map[string]any:
		if len(typed) != 1 {
			return nil, fmt.Errorf("quiet mode requires exactly one projected field")
		}
		for _, value := range typed {
			return []string{stringify(value)}, nil
		}
	case []any:
		lines := make([]string, 0, len(typed))
		for _, item := range typed {
			record, ok := item.(map[string]any)
			if !ok || len(record) != 1 {
				return nil, fmt.Errorf("quiet mode requires exactly one projected field")
			}
			for _, value := range record {
				lines = append(lines, stringify(value))
			}
		}
		return lines, nil
	default:
		return []string{stringify(typed)}, nil
	}
	return nil, fmt.Errorf("quiet mode could not extract values")
}

func toRows(data any) ([]map[string]any, error) {
	generic, err := toGeneric(data)
	if err != nil {
		return nil, err
	}
	items, ok := generic.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list")
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected a list of objects")
		}
		rows = append(rows, record)
	}
	return rows, nil
}

func selectedColumns(columns, defaults []string) []string {
	if len(columns) > 0 {
		return columns
	}
	return defaults
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}
