package parser

import (
	"context"
	"fmt"
	"io"
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
	parser, err := r.GetParser(filePathOrType)
	if err != nil {
		return nil, err
	}

	return parser.Parse(ctx, reader)
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
