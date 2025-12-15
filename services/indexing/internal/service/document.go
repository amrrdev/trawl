package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/amrrdev/trawl/services/shared/storage"
)

const (
	urlExpiryDuration = 15 * time.Minute
)

type Document struct {
	storage *storage.Storage
}

type GetUrlResponse struct {
	PresignedUrl string `json:"pre-signed_url"`
	ValidFor     string `json:"valid_for"`
}

type GetListFileResponse struct {
	Files []map[string]any `json:"files"`
}

func NewDocument(storage *storage.Storage) *Document {
	return &Document{
		storage: storage,
	}
}

func (d *Document) ListFiles(ctx context.Context, userID string) (*GetListFileResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("userID is required")
	}

	files, err := d.storage.ListFiles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return &GetListFileResponse{
		Files: files,
	}, nil
}

func (d *Document) GetDownloadUrl(ctx context.Context, userID, filename string) (*GetUrlResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("userID is required")
	}
	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("filename is required")
	}

	presignedUrl, err := d.storage.GetDownloadUrl(ctx, userID, filename, urlExpiryDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

	return &GetUrlResponse{
		PresignedUrl: presignedUrl,
		ValidFor:     fmt.Sprintf("%.0f minutes", urlExpiryDuration.Minutes()),
	}, nil
}

func (d *Document) GetUploadUrl(ctx context.Context, userID, filename string) (*GetUrlResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("userID is required")
	}
	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("filename is required")
	}

	presignedUrl, err := d.storage.GetUploadUrl(ctx, userID, filename, urlExpiryDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return &GetUrlResponse{
		PresignedUrl: presignedUrl,
		ValidFor:     fmt.Sprintf("%.0f minutes", urlExpiryDuration.Minutes()),
	}, nil
}
