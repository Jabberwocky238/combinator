package common

import (
	"io"
	"time"

	"jabberwocky238/combinator/core/common/models"
)

type Service interface {
	Start() error
	Close() error
	Type() string
}

type RDB interface {
	Service
	Query(stmt string, args ...any) (io.ReadCloser, error)
	Exec(stmt string, args ...any) error
	Batch(stmt []string, args [][]any) error
}

type KV interface {
	Service
	Get(key string, opts *models.KVGetOptions) (io.ReadCloser, error)
	Set(key string, value io.Reader, opts *models.KVSetOptions) error
	Del(key string, opts *models.KVDelOptions) error
}

type Queue interface {
	Service
}

type S3 interface {
	Service

	// 基础操作
	Head(key string) (*models.S3ObjectInfo, error)
	Get(key string, opts *models.S3GetOptions) (io.ReadCloser, *models.S3ObjectInfo, error)
	Put(key string, reader io.Reader, size int64, opts *models.S3PutOptions) error
	Delete(opts *models.S3DeleteOptions) (int, error)
	Copy(srcKey, dstKey string) error

	// 列表操作
	List(opts *models.S3ListOptions) (*models.S3ListResult, error)

	// 预签名URL
	GetPresignedURL(key string, expires time.Duration) (string, error)
	PutPresignedURL(key string, expires time.Duration) (string, error)
}
