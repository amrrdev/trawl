package service

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/amrrdev/trawl/services/indexing/internal/queue"
	"github.com/amrrdev/trawl/services/indexing/internal/types"
	"github.com/amrrdev/trawl/services/shared/storage"
	"github.com/google/uuid"
)

const (
	urlExpiryDuration = 15 * time.Minute
)

type Document struct {
	storage  *storage.Storage
	producer *queue.Producer
}

type GetUrlResponse struct {
	PresignedUrl string `json:"pre-signed_url"`
	ValidFor     string `json:"valid_for"`
}

type GetListFileResponse struct {
	Files []map[string]any `json:"files"`
}

func NewDocument(storage *storage.Storage, producer *queue.Producer) *Document {
	return &Document{
		storage:  storage,
		producer: producer,
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

func (d *Document) HandlerWebhook(ctx context.Context, event *types.MinIOEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	for _, record := range event.Records {
		if record.EventName == "s3:ObjectCreated:Put" {
			log.Printf("File uploaded: %s (size: %d bytes)",
				record.S3.Object.Key,
				record.S3.Object.Size)

			// Decode URL-encoded object key
			decodedKey, err := url.QueryUnescape(record.S3.Object.Key)
			if err != nil {
				log.Printf("Failed to decode object key: %s", record.S3.Object.Key)
				continue
			}

			// Extract userID from object key (format: "userID/filename")
			parts := strings.SplitN(decodedKey, "/", 2)
			if len(parts) != 2 {
				log.Printf("Invalid object key format: %s", decodedKey)
				continue
			}

			userID := parts[0]
			fileName := parts[1]

			// Create indexing job
			job := &types.IndexingJob{
				JobID:     uuid.New().String(),
				Type:      "document_indexing",
				CreatedAt: time.Now(),
				Payload: types.IndexingPayload{
					DocID:    uuid.New().String(),
					UserID:   userID,
					FilePath: decodedKey, // Use decoded key
					FileName: fileName,
					FileSize: record.S3.Object.Size,
					Metadata: map[string]string{
						"bucket": record.S3.Bucket.Name,
					},
				},
				RetryCount: 0,
			}

			if err := d.producer.PublishIndexingJob(ctx, job); err != nil {
				log.Printf("Failed to publish job: %v", err)
				return fmt.Errorf("failed to publish indexing job: %w", err)
			}
		}
	}

	return nil
}
