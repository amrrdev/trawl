package parser

import (
	"context"
	"io"
)

type Parser interface {
	Parse(ctx context.Context, reader io.Reader) (*ParsedDocument, error)
	SupportedTypes() []string
}

type ParsedDocument struct {
	Content  string
	Metadata map[string]string
}
