package service

import (
	"testing"
)

func TestParse_JSON(t *testing.T) {
	p := NewParserService()

	tests := []struct {
		name   string
		input  []byte
		format string
	}{
		{"object", []byte(`{"name":"alice","age":30}`), "json"},
		{"array", []byte(`[1,2,3]`), "json"},
		{"nested", []byte(`{"users":[{"id":1}]}`), "json"},
		{"whitespace", []byte(`  { "key": "value" }  `), "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.input)
			if result.Format != tt.format {
				t.Errorf("expected format %q, got %q", tt.format, result.Format)
			}
			if result.Value == nil {
				t.Error("expected non-nil value")
			}
		})
	}
}

func TestParse_Text(t *testing.T) {
	p := NewParserService()

	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"plain", []byte("hello world"), "hello world"},
		{"number", []byte("42"), "42"},
		{"empty", []byte(""), ""},
		{"unicode", []byte("你好世界"), "你好世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.input)
			if result.Format != "text" {
				t.Errorf("expected format text, got %q", result.Format)
			}
			if s, ok := result.Value.(string); ok && s != tt.want {
				t.Errorf("expected value %q, got %q", tt.want, s)
			}
		})
	}
}

func TestParse_Binary(t *testing.T) {
	p := NewParserService()
	input := []byte{0x00, 0x01, 0x80, 0xFF, 0xFE}

	result := p.Parse(input)
	if result.Format != "binary" {
		t.Errorf("expected format binary, got %q", result.Format)
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	p := NewParserService()
	input := []byte(`{"incomplete":`)

	result := p.Parse(input)
	if result.Format != "text" {
		t.Errorf("expected format text for invalid JSON, got %q", result.Format)
	}
}
