package checkmarx

import (
	"errors"
	"strings"
	"testing"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error should not be retryable",
			err:      nil,
			expected: false,
		},
		{
			name:     "DNS timeout should be retryable",
			err:      errors.New("dial tcp4: lookup deu.ast.checkmarx.net on 127.0.0.53:53: read udp 127.0.0.1:46072->127.0.0.53:53: i/o timeout"),
			expected: true,
		},
		{
			name:     "connection timeout should be retryable",
			err:      errors.New("dial tcp: i/o timeout"),
			expected: true,
		},
		{
			name:     "connection refused should be retryable",
			err:      errors.New("dial tcp: connection refused"),
			expected: true,
		},
		{
			name:     "no such host should be retryable",
			err:      errors.New("dial tcp: no such host"),
			expected: true,
		},
		{
			name:     "network unreachable should be retryable",
			err:      errors.New("dial tcp: network is unreachable"),
			expected: true,
		},
		{
			name:     "temporary DNS failure should be retryable",
			err:      errors.New("dial tcp: temporary failure in name resolution"),
			expected: true,
		},
		{
			name:     "HTTP 401 should not be retryable",
			err:      errors.New("HTTP 401 Unauthorized"),
			expected: false,
		},
		{
			name:     "JSON parsing error should not be retryable",
			err:      errors.New("invalid character 'x' looking for beginning of value"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestRetryOperation(t *testing.T) {
	tests := []struct {
		name          string
		operation     func() error
		expectSuccess bool
		expectRetries bool
	}{
		{
			name: "successful operation should not retry",
			operation: func() error {
				return nil
			},
			expectSuccess: true,
			expectRetries: false,
		},
		{
			name: "retryable error that eventually succeeds",
			operation: func() func() error {
				attempts := 0
				return func() error {
					attempts++
					if attempts < 3 {
						return errors.New("dial tcp4: lookup deu.iam.checkmarx.net on 127.0.0.53:53: read udp 127.0.0.1:49402->127.0.0.53:53: i/o timeout")
					}
					return nil
				}
			}(),
			expectSuccess: true,
			expectRetries: true,
		},
		{
			name: "non-retryable error should fail immediately",
			operation: func() error {
				return errors.New("HTTP 401 Unauthorized")
			},
			expectSuccess: false,
			expectRetries: false,
		},
		{
			name: "retryable error that never succeeds",
			operation: func() error {
				return errors.New("dial tcp: connection refused")
			},
			expectSuccess: false,
			expectRetries: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RetryOperation(tt.operation)
			
			if tt.expectSuccess && err != nil {
				t.Errorf("RetryOperation() expected success but got error: %v", err)
			}
			
			if !tt.expectSuccess && err == nil {
				t.Errorf("RetryOperation() expected failure but got success")
			}
			
			if tt.expectRetries && err != nil && !isRetryableError(err) {
				// For operations that exhaust retries, the final error should mention attempts
				if !strings.Contains(err.Error(), "failed after") {
					t.Errorf("RetryOperation() should indicate retry attempts in error message: %v", err)
				}
			}
		})
	}
}
