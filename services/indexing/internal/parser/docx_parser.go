package parser

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nguyenthenguyen/docx"
)

type DOCXParser struct{}

func NewDOCXParser() *DOCXParser {
	return &DOCXParser{}
}

func (p *DOCXParser) Parse(ctx context.Context, reader io.Reader) (*ParsedDocument, error) {
	tmpFile, err := os.CreateTemp("", "docx-*.docx")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, reader); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	tmpFile.Close()

	doc, err := docx.ReadDocxFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read docx: %w", err)
	}
	defer doc.Close()

	docxText := doc.Editable()
	extractedText := strings.TrimSpace(docxText.GetContent())

	if extractedText == "" {
		return nil, fmt.Errorf("no text content found in DOCX")
	}

	return &ParsedDocument{
		Content: extractedText,
		Metadata: map[string]string{
			"fileType": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
	}, nil
}

func (p *DOCXParser) SupportedTypes() []string {
	return []string{
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".docx",
	}
}
