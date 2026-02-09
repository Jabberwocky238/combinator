package rdb

import (
	"fmt"
	"net/url"
	"runtime"
	"strconv"
	"strings"
)

// ParsedRDBURL contains parsed database connection information
type ParsedRDBURL struct {
	Type     string // "postgres" or "sqlite"
	DSN      string //  Data Source Name
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Path     string // for sqlite file path
}

// ParseRDBURL parses a database URL into connection parameters
// Supports:
//   - postgres://user:pass@host:port/dbname
//   - sqlite:///path/to/db.db
//   - sqlite://:memory:
func ParseRDBURL(rawURL string) (*ParsedRDBURL, error) {
	// 找到第一个 :// 的位置
	idx := strings.Index(rawURL, "://")
	if idx == -1 {
		return nil, fmt.Errorf("invalid RDB URL: %s", rawURL)
	}

	dbType := rawURL[:idx]
	rawInner := rawURL[idx+3:] // 跳过 ://

	fmt.Println("Paring RDB URL: " + rawURL)
	switch dbType {
	case "postgres", "postgresql":
		return parsePostgresURL(dbType, rawInner)
	case "sqlite":
		return parseSQLiteURL(dbType, rawInner)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// parsePostgresURL parses a PostgreSQL URL
func parsePostgresURL(dbType, rawInner string) (*ParsedRDBURL, error) {
	u, err := url.Parse(dbType + "://" + rawInner)
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL URL: %w", err)
	}
	parsed := &ParsedRDBURL{
		Type: "postgres",
		DSN:  dbType + "://" + rawInner,
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
		parsed.Port = 5432 // default postgres port
	}

	// Parse user and password
	if u.User != nil {
		parsed.User = u.User.Username()
		if pass, ok := u.User.Password(); ok {
			parsed.Password = pass
		}
	}

	// Parse database name
	parsed.DBName = strings.TrimPrefix(u.Path, "/")

	return parsed, nil
}

// parseSQLiteURL parses a SQLite URL
func parseSQLiteURL(dbType, rawInner string) (*ParsedRDBURL, error) {
	parsed := &ParsedRDBURL{
		Type: "sqlite",
		DSN:  dbType + "://" + rawInner,
	}

	// Handle :memory: database
	if rawInner == ":memory:" {
		parsed.Path = ":memory:"
		return parsed, nil
	}

	// Handle file path
	// Windows 路径需要转换反斜杠为正斜杠
	if runtime.GOOS == "windows" {
		rawInner = strings.ReplaceAll(rawInner, "\\", "/")
	}
	parsed.Path = rawInner

	return parsed, nil
}
