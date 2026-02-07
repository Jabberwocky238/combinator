package s3

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	common "jabberwocky238/combinator/core/common"
	"jabberwocky238/combinator/core/common/models"
)

func init() {
	RegisterS3Factory("local", func(parsed *ParsedS3URL) (common.S3, error) {
		return NewLocalS3(parsed.Path), nil
	})
}

type LocalS3 struct {
	basePath string
}

func NewLocalS3(basePath string) *LocalS3 {
	return &LocalS3{
		basePath: basePath,
	}
}

func (s *LocalS3) Start() error {
	return os.MkdirAll(s.basePath, 0755)
}

func (s *LocalS3) Close() error {
	return nil
}

func (s *LocalS3) Type() string {
	return "local"
}

func (s *LocalS3) getFullPath(key string) string {
	return filepath.Join(s.basePath, key)
}

// Head 获取对象元数据
func (s *LocalS3) Head(key string) (*models.S3ObjectInfo, error) {
	fullPath := s.getFullPath(key)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return &models.S3ObjectInfo{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
		ContentType:  "application/octet-stream",
	}, nil
}

func (s *LocalS3) Get(key string, opts *models.S3GetOptions) (io.ReadCloser, *models.S3ObjectInfo, error) {
	fullPath := s.getFullPath(key)

	// 先获取文件信息
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	if opts != nil && opts.Range != nil {
		_, err := file.Seek(opts.Range.Start, 0)
		if err != nil {
			file.Close()
			return nil, nil, fmt.Errorf("failed to seek: %w", err)
		}
	}

	info := &models.S3ObjectInfo{
		Key:          key,
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
		ContentType:  "application/octet-stream",
	}

	return file, info, nil
}

func (s *LocalS3) Put(key string, reader io.Reader, size int64, opts *models.S3PutOptions) error {
	fullPath := s.getFullPath(key)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func (s *LocalS3) List(opts *models.S3ListOptions) (*models.S3ListResult, error) {
	result := &models.S3ListResult{
		Objects:  make([]models.S3ObjectInfo, 0),
		Prefixes: make([]string, 0),
	}

	prefix := ""
	if opts != nil {
		prefix = opts.Prefix
	}
	prefixPath := s.getFullPath(prefix)

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(s.basePath, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if prefix == "" || strings.HasPrefix(s.getFullPath(relPath), prefixPath) {
			result.Objects = append(result.Objects, models.S3ObjectInfo{
				Key:          relPath,
				Size:         info.Size(),
				LastModified: info.ModTime(),
				ContentType:  "application/octet-stream",
			})
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	return result, nil
}

func (s *LocalS3) Delete(opts *models.S3DeleteOptions) (int, error) {
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

	// 批量删除
	deletedCount := 0
	for _, key := range keysToDelete {
		fullPath := s.getFullPath(key)
		if err := os.Remove(fullPath); err != nil {
			if !os.IsNotExist(err) {
				return deletedCount, fmt.Errorf("failed to delete file %s: %w", key, err)
			}
		} else {
			deletedCount++
		}
	}

	return deletedCount, nil
}

// Copy 复制对象
func (s *LocalS3) Copy(srcKey, dstKey string) error {
	srcPath := s.getFullPath(srcKey)
	dstPath := s.getFullPath(dstKey)

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(dstPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}
	return nil
}

func (s *LocalS3) GetPresignedURL(key string, expires time.Duration) (string, error) {
	return "", fmt.Errorf("presigned URLs not supported for local storage")
}

// PutPresignedURL 生成预签名上传URL（本地存储不支持）
func (s *LocalS3) PutPresignedURL(key string, expires time.Duration) (string, error) {
	return "", fmt.Errorf("presigned URLs not supported for local storage")
}
