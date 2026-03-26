package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

type ParsedValue struct {
	Value  interface{} `json:"value"`
	Format string      `json:"format"`
}

type ParserService struct{}

func NewParserService() *ParserService {
	return &ParserService{}
}

// Parse auto-detects the format of raw Redis bytes and returns
// a structured representation with format metadata.
func (s *ParserService) Parse(raw []byte) ParsedValue {
	if len(raw) == 0 {
		return ParsedValue{Value: "", Format: "text"}
	}

	trimmed := strings.TrimSpace(string(raw))

	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		var v interface{}
		if err := json.Unmarshal([]byte(trimmed), &v); err == nil {
			return ParsedValue{Value: v, Format: "json"}
		}
	}

	if utf8.Valid(raw) {
		return ParsedValue{Value: string(raw), Format: "text"}
	}

	return ParsedValue{
		Value:  fmt.Sprintf("(binary data, %d bytes)", len(raw)),
		Format: "binary",
	}
}
