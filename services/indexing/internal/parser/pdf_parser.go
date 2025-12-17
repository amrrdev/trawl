package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
)

type PDFParser struct{}

func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

func (p *PDFParser) Parse(ctx context.Context, reader io.Reader) (*ParsedDocument, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read the pdf: %w", err)
	}

	readAt := bytes.NewReader(data)

	pdfReader, err := pdf.NewReader(readAt, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse pdf: %w", err)
	}

	numPages := pdfReader.NumPage()
	var textBuilder strings.Builder

	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	extractedText := strings.TrimSpace(textBuilder.String())
	if extractedText == "" {
		return nil, fmt.Errorf("no text content found in PDF")
	}

	return &ParsedDocument{
		Content: extractedText,
		Metadata: map[string]string{
			"pages":    strconv.Itoa(numPages),
			"fileType": "application/pdf",
		},
	}, nil
}

func (p *PDFParser) SupportedTypes() []string {
	return []string{"application/pdf", ".pdf"}
}
