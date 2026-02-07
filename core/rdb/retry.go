package rdb

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	common "jabberwocky238/combinator/core/common"
)

const (
	maxRetries     = 3
	retryDelay     = 100 * time.Millisecond
	maxRetryDelay  = 2 * time.Second
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

// retryWithReconnect executes a function with retry logic and reconnection
func retryWithReconnect(operation func() error, reconnect func() error, opName string) error {
	var lastErr error
	delay := retryDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			common.Logger.Warnf("[%s] Retry attempt %d/%d after %v", opName, attempt, maxRetries, delay)
			time.Sleep(delay)

			// Exponential backoff
			delay *= 2
			if delay > maxRetryDelay {
				delay = maxRetryDelay
			}

			// Try to reconnect before retry
			if err := reconnect(); err != nil {
				common.Logger.Errorf("[%s] Reconnection failed: %v", opName, err)
				lastErr = err
				continue
			}
			common.Logger.Infof("[%s] Reconnection successful", opName)
		}

		// Execute the operation
		err := operation()
		if err == nil {
			if attempt > 0 {
				common.Logger.Infof("[%s] Operation succeeded after %d retries", opName, attempt)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			common.Logger.Debugf("[%s] Non-retryable error: %v", opName, err)
			return err
		}

		common.Logger.Warnf("[%s] Retryable error detected: %v", opName, err)
	}

	return ebcore.Error("[%s] Failed after %d retries: %v", opName, maxRetries, lastErr)
}
