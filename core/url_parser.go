package combinator

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParsedRDBURL contains parsed database connection information
type ParsedRDBURL struct {
	Type     string // "postgres" or "sqlite"
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
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Scheme {
	case "postgres":
		return parsePostgresURL(u)
	case "sqlite":
		return parseSQLiteURL(u)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", u.Scheme)
	}
}

// parsePostgresURL parses a PostgreSQL URL
func parsePostgresURL(u *url.URL) (*ParsedRDBURL, error) {
	parsed := &ParsedRDBURL{
		Type: "postgres",
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
func parseSQLiteURL(u *url.URL) (*ParsedRDBURL, error) {
	parsed := &ParsedRDBURL{
		Type: "sqlite",
	}

	// Handle :memory: database
	if u.Host == ":memory:" || u.Path == ":memory:" {
		parsed.Path = ":memory:"
		return parsed, nil
	}

	// Handle file path
	// sqlite:///path/to/db.db -> /path/to/db.db
	// sqlite://path/to/db.db -> path/to/db.db
	if u.Host == "" {
		parsed.Path = u.Path
	} else {
		parsed.Path = u.Host + u.Path
	}

	return parsed, nil
}
