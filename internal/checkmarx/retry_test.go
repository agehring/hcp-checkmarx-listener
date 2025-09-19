package checkmarx

import (
	"errors"
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
