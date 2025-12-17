package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type JSONParser struct{}

func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

func (p *JSONParser) Parse(ctx context.Context, reader io.Reader) (*ParsedDocument, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	var textBuilder strings.Builder
	extractText(jsonData, &textBuilder)

	content := strings.TrimSpace(textBuilder.String())
	if content == "" {
		return nil, fmt.Errorf("no text content found in JSON")
	}

	return &ParsedDocument{
		Content: content,
		Metadata: map[string]string{
			"fileType": "application/json",
		},
	}, nil
}

func (p *JSONParser) SupportedTypes() []string {
	return []string{"application/json", ".json"}
}

func extractText(data interface{}, builder *strings.Builder) {
	switch v := data.(type) {
	case string:
		builder.WriteString(v)
		builder.WriteString(" ")
	case map[string]interface{}:
		for _, value := range v {
			extractText(value, builder)
		}
	case []interface{}:
		for _, item := range v {
			extractText(item, builder)
		}
	}
}
