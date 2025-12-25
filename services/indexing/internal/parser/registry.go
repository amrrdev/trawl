package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
)

type Registry struct {
	parsers map[string]Parser
}

func NewRegistry() *Registry {
	registry := &Registry{
		parsers: make(map[string]Parser),
	}

	registry.Register(NewTextParser())
	registry.Register(NewJSONParser())
	registry.Register(NewPDFParser())
	registry.Register(NewDOCXParser())

	return registry
}

func (r *Registry) Register(parser Parser) {
	for _, contentType := range parser.SupportedTypes() {
		r.parsers[strings.ToLower(contentType)] = parser
	}
}

func (r *Registry) GetParser(filePathOrType string) (Parser, error) {
	ext := strings.ToLower(filepath.Ext(filePathOrType))
	if parser, ok := r.parsers[ext]; ok {
		return parser, nil
	}

	contentType := strings.ToLower(filePathOrType)
	if parser, ok := r.parsers[contentType]; ok {
		return parser, nil
	}

	return nil, fmt.Errorf("unsupported file type: %s", filePathOrType)
}

func (r *Registry) ParseFile(ctx context.Context, filePathOrType string, reader io.Reader) (*ParsedDocument, error) {
	// Read the content first to enable multiple parsing attempts
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// Try content-based detection first
	if len(data) >= 4 && bytes.HasPrefix(data, []byte("%PDF")) {
		contentReader := bytes.NewReader(data)
		if parser, ok := r.parsers[".pdf"]; ok {
			result, err := parser.Parse(ctx, contentReader)
			if err == nil {
				return result, nil
			}
		}
	}

	// Try based on file extension
	ext := strings.ToLower(filepath.Ext(filePathOrType))
	if parser, ok := r.parsers[ext]; ok {
		contentReader := bytes.NewReader(data)
		result, err := parser.Parse(ctx, contentReader)
		if err == nil {
			return result, nil
		}
		// Log the failure but continue trying other parsers
		log.Printf("⚠️  Failed to parse %s as %s: %v", filePathOrType, ext, err)
	}

	// Try all other parsers as fallback
	for contentType, parser := range r.parsers {
		if contentType == ext {
			continue // Already tried this one
		}

		contentReader := bytes.NewReader(data)
		result, err := parser.Parse(ctx, contentReader)
		if err == nil {
			log.Printf("⚠️  File %s parsed as %s instead of expected %s", filePathOrType, contentType, ext)
			return result, nil
		}
	}

	return nil, fmt.Errorf("failed to parse file %s with any available parser", filePathOrType)
}

func (r *Registry) SupportedTypes() []string {
	var types []string
	seen := make(map[string]bool)

	for contentType := range r.parsers {
		if !seen[contentType] {
			types = append(types, contentType)
			seen[contentType] = true
		}
	}

	return types
}
