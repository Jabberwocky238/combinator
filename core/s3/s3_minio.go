package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	common "jabberwocky238/combinator/core/common"
)

func init() {
	RegisterS3Factory("minio", func(parsed *ParsedS3URL) (common.S3, error) {
		return NewMinioS3(parsed)
	})
}

type MinioS3 struct {
	client *minio.Client
	bucket string
	config *ParsedS3URL
	ctx    context.Context
}

func NewMinioS3(parsed *ParsedS3URL) (*MinioS3, error) {
	endpoint := parsed.Host
	if parsed.Port != "" {
		endpoint = fmt.Sprintf("%s:%s", parsed.Host, parsed.Port)
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(parsed.AccessKey, parsed.SecretKey, ""),
		Secure: parsed.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinioS3{
		client: client,
		bucket: parsed.Bucket,
		config: parsed,
		ctx:    context.Background(),
	}, nil
}

func (s *MinioS3) Start() error {
	exists, err := s.client.BucketExists(s.ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(s.ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

func (s *MinioS3) Close() error {
	return nil
}

func (s *MinioS3) Type() string {
	return "minio"
}

func (s *MinioS3) Get(key string) ([]byte, error) {
	obj, err := s.client.GetObject(s.ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}
	return data, nil
}

func (s *MinioS3) Put(key string, value []byte) error {
	reader := bytes.NewReader(value)
	_, err := s.client.PutObject(s.ctx, s.bucket, key, reader, int64(len(value)), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (s *MinioS3) List(prefix string) ([]string, error) {
	var keys []string

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for obj := range s.client.ListObjects(s.ctx, s.bucket, opts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("failed to list: %w", obj.Err)
		}
		keys = append(keys, obj.Key)
	}
	return keys, nil
}

func (s *MinioS3) Delete(key string) error {
	err := s.client.RemoveObject(s.ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

func (s *MinioS3) GeneratePresignedUploadURL(key string) (string, error) {
	url, err := s.client.PresignedPutObject(s.ctx, s.bucket, key, time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}
	return url.String(), nil
}

func (s *MinioS3) GeneratePresignedDownloadURL(key string) (string, error) {
	url, err := s.client.PresignedGetObject(s.ctx, s.bucket, key, time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}
	return url.String(), nil
}
