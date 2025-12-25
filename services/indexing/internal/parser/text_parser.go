package parser

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type TextParser struct{}

func NewTextParser() *TextParser {
	return &TextParser{}
}

func (p *TextParser) Parse(ctx context.Context, reader io.Reader) (*ParsedDocument, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read text file: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil, fmt.Errorf("no text content found")
	}

	return &ParsedDocument{
		Content: content,
		Metadata: map[string]string{
			"fileType": "text/plain",
		},
	}, nil
}

func (p *TextParser) SupportedTypes() []string {
	return []string{"text/plain", ".txt", ".log", ".md", ".csv", ".pdf"}
}
