package parser

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/rsc/pdf"
)

type PDFParser struct{}

func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *PDFParser) Parse(ctx context.Context, reader io.Reader) (*ParsedDocument, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read the pdf: %w", err)
	}

	// Check if it's a valid PDF by examining the header
	if len(data) < 4 || !strings.HasPrefix(string(data), "%PDF") {
		return nil, fmt.Errorf("not a PDF file: invalid header (got: %q)", string(data[:min(10, len(data))]))
	}

	// Parse the PDF
	r, err := pdf.NewReader(strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse pdf: %w", err)
	}

	var textBuilder strings.Builder
	numPages := r.NumPage()

	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		content := page.Content()
		for _, text := range content.Text {
			textBuilder.WriteString(text.S)
			textBuilder.WriteString(" ")
		}
		textBuilder.WriteString("\n")
	}

	extractedText := strings.TrimSpace(textBuilder.String())
	if extractedText == "" {
		return nil, fmt.Errorf("no text content found in PDF")
	}

	return &ParsedDocument{
		Content: extractedText,
		Metadata: map[string]string{
			"pages":    fmt.Sprintf("%d", numPages),
			"fileType": "application/pdf",
		},
	}, nil
}

func (p *PDFParser) SupportedTypes() []string {
	return []string{"application/pdf", ".pdf"}
}
