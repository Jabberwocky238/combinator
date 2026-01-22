package kv

import (
	"fmt"
	"net/url"
	"strconv"
)

// ParsedKVURL contains parsed KV store connection information
type ParsedKVURL struct {
	Type     string // "redis", "rocksdb", or "memory"
	Host     string
	Port     int
	Password string
	DB       int    // for redis database number
	Path     string // for rocksdb file path
}

// ParseKVURL parses a KV store URL into connection parameters
// Supports:
//   - redis://[:password@]host:port[/db]
//   - rocksdb:///path/to/db
//   - memory://
func ParseKVURL(rawURL string) (*ParsedKVURL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Scheme {
	case "redis":
		return parseRedisURL(u)
	case "rocksdb":
		return parseRocksDBURL(u)
	case "memory":
		return parseMemoryURL(u)
	default:
		return nil, fmt.Errorf("unsupported KV store type: %s", u.Scheme)
	}
}

// parseRedisURL parses a Redis URL
func parseRedisURL(u *url.URL) (*ParsedKVURL, error) {
	parsed := &ParsedKVURL{
		Type: "redis",
		Host: u.Hostname(),
	}

	// Parse port
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
		parsed.Port = port
	} else {
		parsed.Port = 6379 // default redis port
	}

	// Parse password
	if u.User != nil {
		if pass, ok := u.User.Password(); ok {
			parsed.Password = pass
		}
	}

	// Parse database number
	if u.Path != "" && u.Path != "/" {
		dbStr := u.Path[1:] // remove leading /
		db, err := strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid database number: %w", err)
		}
		parsed.DB = db
	}

	return parsed, nil
}

// parseRocksDBURL parses a RocksDB URL
func parseRocksDBURL(u *url.URL) (*ParsedKVURL, error) {
	parsed := &ParsedKVURL{
		Type: "rocksdb",
	}

	// Handle :memory: database
	if u.Host == ":memory:" || u.Path == ":memory:" {
		parsed.Path = ":memory:"
		return parsed, nil
	}

	// Handle file path
	// rocksdb:///path/to/db -> /path/to/db
	// rocksdb://path/to/db -> path/to/db
	if u.Host == "" {
		parsed.Path = u.Path
	} else {
		parsed.Path = u.Host + u.Path
	}

	return parsed, nil
}

// parseMemoryURL parses a Memory URL
func parseMemoryURL(u *url.URL) (*ParsedKVURL, error) {
	return &ParsedKVURL{
		Type: "memory",
	}, nil
}
