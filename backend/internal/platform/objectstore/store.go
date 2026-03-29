package objectstore

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"tramplin/backend/internal/config"
)

var ErrObjectNotFound = errors.New("object not found")

type Store interface {
	PresignUpload(ctx context.Context, objectKey, contentType string, expires time.Duration) (*PresignedUpload, error)
	PresignDownload(ctx context.Context, objectKey string, expires time.Duration) (*PresignedDownload, error)
	StatObject(ctx context.Context, objectKey string) (*ObjectInfo, error)
	DeleteObject(ctx context.Context, objectKey string) error
}

type PresignedUpload struct {
	URL       string
	Method    string
	Headers   map[string]string
	ExpiresAt time.Time
}

type PresignedDownload struct {
	URL       string
	ExpiresAt time.Time
}

type ObjectInfo struct {
	ETag        string
	Size        int64
	ContentType string
}

type minioStore struct {
	client *minio.Client
	bucket string
}

func NewMinIO(ctx context.Context, cfg config.Config) (Store, error) {
	client, err := minio.New(cfg.ObjectStorageEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.ObjectStorageAccessKey, cfg.ObjectStorageSecretKey, ""),
		Secure: cfg.ObjectStorageUseSSL,
		Region: cfg.ObjectStorageRegion,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.ObjectStorageBucket)
	if err != nil {
		return nil, fmt.Errorf("check object storage bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.ObjectStorageBucket, minio.MakeBucketOptions{
			Region: cfg.ObjectStorageRegion,
		}); err != nil {
			return nil, fmt.Errorf("create object storage bucket: %w", err)
		}
	}

	return &minioStore{
		client: client,
		bucket: cfg.ObjectStorageBucket,
	}, nil
}

func (s *minioStore) PresignUpload(ctx context.Context, objectKey, _ string, expires time.Duration) (*PresignedUpload, error) {
	presignedURL, err := s.client.PresignedPutObject(ctx, s.bucket, objectKey, expires)
	if err != nil {
		return nil, fmt.Errorf("presign upload: %w", err)
	}
	return &PresignedUpload{
		URL:       presignedURL.String(),
		Method:    "PUT",
		Headers:   map[string]string{},
		ExpiresAt: time.Now().UTC().Add(expires),
	}, nil
}

func (s *minioStore) PresignDownload(ctx context.Context, objectKey string, expires time.Duration) (*PresignedDownload, error) {
	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expires, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("presign download: %w", err)
	}
	return &PresignedDownload{
		URL:       presignedURL.String(),
		ExpiresAt: time.Now().UTC().Add(expires),
	}, nil
}

func (s *minioStore) StatObject(ctx context.Context, objectKey string) (*ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		if isObjectNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("stat object: %w", err)
	}
	return &ObjectInfo{
		ETag:        info.ETag,
		Size:        info.Size,
		ContentType: info.ContentType,
	}, nil
}

func (s *minioStore) DeleteObject(ctx context.Context, objectKey string) error {
	err := s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil && !isObjectNotFound(err) {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func isObjectNotFound(err error) bool {
	response := minio.ToErrorResponse(err)
	return response.StatusCode == 404 || response.Code == "NoSuchKey" || response.Code == "NoSuchBucket"
}
