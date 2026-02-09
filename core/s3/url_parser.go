package s3

import (
	"fmt"
	"net/url"
	"runtime"
	"strings"
)

// ParsedS3URL contains parsed S3 store connection information
type ParsedS3URL struct {
	Type      string // "local", "s3", "minio"
	Path      string // for local file path
	Bucket    string
	Region    string
	Host      string
	Port      string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// ParseS3URL parses a S3 store URL into connection parameters
// Supports:
//   - local:///path/to/storage
//   - minio://bucket@host:port
func ParseS3URL(rawURL string) (*ParsedS3URL, error) {
	// 找到第一个 :// 的位置
	idx := strings.Index(rawURL, "://")
	if idx == -1 {
		return nil, fmt.Errorf("invalid RDB URL: %s", rawURL)
	}

	s3Type := rawURL[:idx]
	rawInner := rawURL[idx+3:] // 跳过 ://

	switch s3Type {
	case "local":
		return parseLocalURL(rawInner)
	case "minio":
		return parseMinioURL(rawInner)
	default:
		return nil, fmt.Errorf("unsupported S3 store type: %s", s3Type)
	}
}

// parseLocalURL parses a local storage URL
func parseLocalURL(rawInner string) (*ParsedS3URL, error) {
	parsed := &ParsedS3URL{
		Type: "local",
	}

	// Handle file path
	// local:///path/to/storage -> /path/to/storage
	// local://path/to/storage -> path/to/storage
	if runtime.GOOS == "windows" {
		rawInner = strings.ReplaceAll(rawInner, "\\", "/")
	}
	parsed.Path = rawInner
	return parsed, nil
}

// parseMinioURL parses a MinIO URL
// Format: minio://accessKey:secretKey@host:port/bucket?ssl=true
func parseMinioURL(rawInner string) (*ParsedS3URL, error) {
	u, err := url.Parse("minio://" + rawInner)
	if err != nil {
		return nil, fmt.Errorf("invalid MinIO URL: %w", err)
	}
	parsed := &ParsedS3URL{
		Type: "minio",
		Host: u.Hostname(),
		Port: u.Port(),
	}

	// 解析认证信息
	if u.User != nil {
		parsed.AccessKey = u.User.Username()
		if secret, ok := u.User.Password(); ok {
			parsed.SecretKey = secret
		}
	}

	// 解析 bucket (从 path 获取)
	if u.Path != "" && u.Path != "/" {
		parsed.Bucket = u.Path[1:] // 移除开头的 /
	}

	// 解析 SSL 参数
	parsed.UseSSL = u.Query().Get("ssl") == "true"

	if parsed.Host == "" {
		return nil, fmt.Errorf("MinIO host is required")
	}
	if parsed.Bucket == "" {
		return nil, fmt.Errorf("MinIO bucket is required")
	}

	return parsed, nil
}
