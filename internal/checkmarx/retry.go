package checkmarx

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/httpclient"
	log "github.com/sirupsen/logrus"
)

// isRetryableError checks if an error is retryable for network requests
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// Check for common network timeout and DNS issues
	return strings.Contains(errMsg, "dial tcp") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "network is unreachable") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "temporary failure in name resolution")
}

// retryHTTPRequest performs an HTTP request with retry logic for network issues
func retryHTTPRequest(req *http.Request, body []byte, maxRetries int, baseDelay time.Duration) (*http.Response, error) {
	debug := os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "true"
	logger := log.New()
	if debug {
		logger.SetLevel(log.TraceLevel)
	}

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			if delay > 30*time.Second {
				delay = 30 * time.Second // Cap at 30 seconds
			}

			if debug {
				logger.Infof("Network retry attempt %d/%d after %v delay for %s", attempt, maxRetries, delay, req.URL.String())
			} else {
				fmt.Printf("Network retry attempt %d/%d after %v delay for %s\n", attempt, maxRetries, delay, req.URL.String())
			}

			time.Sleep(delay)
		}

		// Create fresh request for each attempt
		reqClone := req.Clone(req.Context())
		if body != nil {
			reqClone.Body = io.NopCloser(bytes.NewReader(body))
		}

		resp, err := httpclient.Client.Do(reqClone)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Only retry if it's a retryable network error
		if !isRetryableError(err) {
			if debug {
				logger.Errorf("Non-retryable error, failing immediately: %s", err)
			}
			return nil, err
		}

		if debug {
			logger.Warnf("Retryable network error on attempt %d: %s", attempt+1, err)
		} else {
			fmt.Printf("Network error on attempt %d: %s\n", attempt+1, err)
		}
	}

	// All retries exhausted
	if debug {
		logger.Errorf("All %d retry attempts exhausted, final error: %s", maxRetries+1, lastErr)
	}
	return nil, fmt.Errorf("network request failed after %d attempts: %w", maxRetries+1, lastErr)
}

// RetryOperation retries any operation that might fail due to network issues
func RetryOperation(operation func() error) error {
	debug := os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "true"
	logger := log.New()
	if debug {
		logger.SetLevel(log.TraceLevel)
	}

	maxRetries := 3
	baseDelay := 2 * time.Second
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			if delay > 30*time.Second {
				delay = 30 * time.Second // Cap at 30 seconds
			}

			if debug {
				logger.Infof("Operation retry attempt %d/%d after %v delay", attempt, maxRetries, delay)
			} else {
				fmt.Printf("Operation retry attempt %d/%d after %v delay\n", attempt, maxRetries, delay)
			}

			time.Sleep(delay)
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Only retry if it's a retryable network error
		if !isRetryableError(err) {
			if debug {
				logger.Errorf("Non-retryable error, failing immediately: %s", err)
			}
			return err
		}

		if debug {
			logger.Warnf("Retryable network error on attempt %d: %s", attempt+1, err)
		} else {
			fmt.Printf("Network error on attempt %d: %s\n", attempt+1, err)
		}
	}

	// All retries exhausted
	if debug {
		logger.Errorf("All %d retry attempts exhausted, final error: %s", maxRetries+1, lastErr)
	}
	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries+1, lastErr)
}
