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
	Client *minio.Client
	Bucket string
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
		Client: client,
		Bucket: config.Bucket,
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
	presignedUrl, err := s.Client.PresignedPutObject(
		ctx,
		s.Bucket,
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
	presignedUrl, err := s.Client.PresignedGetObject(
		ctx,
		s.Bucket,
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
	objects := s.Client.ListObjects(
		ctx,
		s.Bucket,
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
