package client

import (
	"io"

	"github.com/sixers/fakturownia-cli/internal/jsoninput"
)

func ParseInput(raw string, stdin io.Reader) (map[string]any, error) {
	return jsoninput.ParseObject(raw, stdin, "client")
}
