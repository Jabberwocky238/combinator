package rdb

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	maxRetries    = 3
	retryDelay    = 100 * time.Millisecond
	maxRetryDelay = 2 * time.Second
)

// isRetryableError checks if the error is a connection-related error that should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Common connection errors
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no connection",
		"connection lost",
		"bad connection",
		"driver: bad connection",
		"invalid connection",
		"connection closed",
		"connection timeout",
		"i/o timeout",
		"network is unreachable",
		"connection reset by peer",
		"eof",
		"unexpected eof",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errMsg, retryable) {
			return true
		}
	}

	// Check for sql.ErrConnDone
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}

	return false
}
