package services

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/config"
)

type StorageService struct {
	client *minio.Client
	bucket string
}

func NewStorageService(cfg config.MinIOConfig) (*StorageService, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	return &StorageService{client: client, bucket: cfg.Bucket}, nil
}

func (s *StorageService) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func (s *StorageService) Upload(ctx context.Context, objectPath string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectPath, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *StorageService) PresignedURL(ctx context.Context, objectPath string, ttl time.Duration) (*url.URL, error) {
	return s.client.PresignedGetObject(ctx, s.bucket, objectPath, ttl, nil)
}

func (s *StorageService) Delete(ctx context.Context, objectPath string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectPath, minio.RemoveObjectOptions{})
}
