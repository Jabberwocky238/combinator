package s3

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	common "jabberwocky238/combinator/core/common"
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

func (s *LocalS3) Get(key string) ([]byte, error) {
	fullPath := s.getFullPath(key)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}

func (s *LocalS3) Put(key string, value []byte) error {
	fullPath := s.getFullPath(key)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(fullPath, value, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func (s *LocalS3) List(prefix string) ([]string, error) {
	var keys []string
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
			keys = append(keys, relPath)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	return keys, nil
}

func (s *LocalS3) Delete(key string) error {
	fullPath := s.getFullPath(key)
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (s *LocalS3) GeneratePresignedUploadURL(key string) (string, error) {
	return "", fmt.Errorf("presigned URLs not supported for local storage")
}

func (s *LocalS3) GeneratePresignedDownloadURL(key string) (string, error) {
	return "", fmt.Errorf("presigned URLs not supported for local storage")
}
