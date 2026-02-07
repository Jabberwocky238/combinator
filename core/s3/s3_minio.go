package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	common "jabberwocky238/combinator/core/common"
	"jabberwocky238/combinator/core/common/models"
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

// Head 获取对象元数据
func (s *MinioS3) Head(key string) (*models.S3ObjectInfo, error) {
	stat, err := s.client.StatObject(s.ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	return &models.S3ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		LastModified: stat.LastModified,
		ETag:         stat.ETag,
		ContentType:  stat.ContentType,
		Metadata:     stat.UserMetadata,
	}, nil
}

func (s *MinioS3) Get(key string, opts *models.S3GetOptions) (io.ReadCloser, *models.S3ObjectInfo, error) {
	getOpts := minio.GetObjectOptions{}

	if opts != nil && opts.Range != nil {
		getOpts.SetRange(opts.Range.Start, opts.Range.End)
	}

	obj, err := s.client.GetObject(s.ctx, s.bucket, key, getOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}

	// 获取对象元数据
	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, nil, fmt.Errorf("failed to stat object: %w", err)
	}

	info := &models.S3ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		LastModified: stat.LastModified,
		ETag:         stat.ETag,
		ContentType:  stat.ContentType,
		Metadata:     stat.UserMetadata,
	}

	return obj, info, nil
}

func (s *MinioS3) Put(key string, reader io.Reader, size int64, opts *models.S3PutOptions) error {
	putOpts := minio.PutObjectOptions{}

	if opts != nil {
		if opts.ContentType != "" {
			putOpts.ContentType = opts.ContentType
		}
		if opts.Metadata != nil {
			putOpts.UserMetadata = opts.Metadata
		}
	}

	_, err := s.client.PutObject(s.ctx, s.bucket, key, reader, size, putOpts)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (s *MinioS3) List(opts *models.S3ListOptions) (*models.S3ListResult, error) {
	result := &models.S3ListResult{
		Objects:  make([]models.S3ObjectInfo, 0),
		Prefixes: make([]string, 0),
	}

	listOpts := minio.ListObjectsOptions{
		Recursive: true,
	}

	if opts != nil {
		listOpts.Prefix = opts.Prefix
		listOpts.MaxKeys = opts.MaxKeys
		listOpts.StartAfter = opts.StartAfter
		if opts.Delimiter != "" {
			listOpts.Recursive = false
			listOpts.UseV1 = true
		}
	}

	for obj := range s.client.ListObjects(s.ctx, s.bucket, listOpts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("failed to list: %w", obj.Err)
		}

		result.Objects = append(result.Objects, models.S3ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ETag:         obj.ETag,
			ContentType:  obj.ContentType,
		})
	}

	return result, nil
}

func (s *MinioS3) Delete(opts *models.S3DeleteOptions) (int, error) {
	if opts == nil || len(opts.Keys) == 0 {
		return 0, fmt.Errorf("delete options or keys list is empty")
	}

	// 收集所有需要删除的键
	var keysToDelete []string
	for _, deleteKey := range opts.Keys {
		switch deleteKey.Mode {
		case models.S3DeleteModePrecise:
			// 精确匹配模式，直接添加
			keysToDelete = append(keysToDelete, deleteKey.Key)
		case models.S3DeleteModePrefix:
			// 前缀匹配模式，需要先列出所有匹配的对象
			listOpts := &models.S3ListOptions{
				Prefix: deleteKey.Key,
			}
			result, err := s.List(listOpts)
			if err != nil {
				return 0, fmt.Errorf("failed to list objects with prefix: %w", err)
			}
			for _, obj := range result.Objects {
				keysToDelete = append(keysToDelete, obj.Key)
			}
		default:
			return 0, fmt.Errorf("invalid delete mode: %s", deleteKey.Mode)
		}
	}

	if len(keysToDelete) == 0 {
		return 0, nil
	}

	// 批量删除
	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for _, key := range keysToDelete {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	removeOpts := minio.RemoveObjectsOptions{}
	deletedCount := 0
	for err := range s.client.RemoveObjects(s.ctx, s.bucket, objectsCh, removeOpts) {
		if err.Err != nil {
			return deletedCount, fmt.Errorf("failed to delete %s: %w", err.ObjectName, err.Err)
		}
		deletedCount++
	}

	return deletedCount, nil
}

// Copy 复制对象
func (s *MinioS3) Copy(srcKey, dstKey string) error {
	src := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: srcKey,
	}
	dst := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: dstKey,
	}

	_, err := s.client.CopyObject(s.ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}
	return nil
}

func (s *MinioS3) GetPresignedURL(key string, expires time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(s.ctx, s.bucket, key, expires, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}
	return url.String(), nil
}

// PutPresignedURL 生成预签名上传URL
func (s *MinioS3) PutPresignedURL(key string, expires time.Duration) (string, error) {
	url, err := s.client.PresignedPutObject(s.ctx, s.bucket, key, expires)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}
	return url.String(), nil
}
