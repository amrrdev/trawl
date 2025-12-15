package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client *minio.Client
	bucket string
}

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func NewStorage(ctx context.Context, config *Config) (*Storage, error) {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	s := &Storage{
		client: client,
		bucket: config.Bucket,
	}

	exists, err := client.BucketExists(ctx, config.Bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		err = client.MakeBucket(ctx, config.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Storage) GetUploadUrl(ctx context.Context, userID, filename string, duration time.Duration) (string, error) {
	objectName := GetObjectName(userID, filename)
	presignedUrl, err := s.client.PresignedPutObject(
		ctx,
		s.bucket,
		objectName,
		duration,
	)
	if err != nil {
		return "", err
	}

	return presignedUrl.String(), nil
}

func (s *Storage) GetDownloadUrl(ctx context.Context, userID, filename string, duration time.Duration) (string, error) {
	objectName := GetObjectName(userID, filename)
	presignedUrl, err := s.client.PresignedGetObject(
		ctx,
		s.bucket,
		objectName,
		duration,
		url.Values{},
	)
	if err != nil {
		return "", err
	}

	return presignedUrl.String(), nil
}

func (s *Storage) ListFiles(ctx context.Context, userID string) ([]map[string]any, error) {
	objects := s.client.ListObjects(
		ctx,
		s.bucket,
		minio.ListObjectsOptions{Prefix: userID + "/"},
	)

	var files []map[string]any
	for obj := range objects {
		if obj.Err != nil {
			return nil, obj.Err
		}

		files = append(files, gin.H{
			"name":     obj.Key,
			"size":     obj.Size,
			"modified": obj.LastModified,
		})
	}

	return files, nil
}

func GetObjectName(userID string, filename string) string {
	objectName := fmt.Sprintf("%s/%s", userID, filename)
	return objectName
}
