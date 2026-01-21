package components

import (
	"bytes"
	"encoding/json"
)

func ToJSON(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		return "[]"
	}
	// Encode adds a newline at the end, trim it
	return string(bytes.TrimSpace(buf.Bytes()))
}
