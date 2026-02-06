package s3

import (
	"fmt"
	"net/url"
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
//   - s3://bucket@region
//   - minio://bucket@host:port
func ParseS3URL(rawURL string) (*ParsedS3URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Scheme {
	case "local":
		return parseLocalURL(u)
	case "s3":
		return parseS3URL(u)
	case "minio":
		return parseMinioURL(u)
	default:
		return nil, fmt.Errorf("unsupported S3 store type: %s", u.Scheme)
	}
}

// parseLocalURL parses a local storage URL
func parseLocalURL(u *url.URL) (*ParsedS3URL, error) {
	parsed := &ParsedS3URL{
		Type: "local",
	}

	// Handle file path
	// local:///path/to/storage -> /path/to/storage
	// local://path/to/storage -> path/to/storage
	if u.Host == "" {
		parsed.Path = u.Path
	} else {
		parsed.Path = u.Host + u.Path
	}

	if parsed.Path == "" {
		return nil, fmt.Errorf("local storage path is required")
	}

	return parsed, nil
}

// parseS3URL parses an AWS S3 URL
func parseS3URL(u *url.URL) (*ParsedS3URL, error) {
	parsed := &ParsedS3URL{
		Type:   "s3",
		Bucket: u.Host,
		Region: u.User.Username(),
	}

	if parsed.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket is required")
	}

	return parsed, nil
}

// parseMinioURL parses a MinIO URL
// Format: minio://accessKey:secretKey@host:port/bucket?ssl=true
func parseMinioURL(u *url.URL) (*ParsedS3URL, error) {
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
