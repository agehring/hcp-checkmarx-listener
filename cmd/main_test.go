package main

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTeapotResponses(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		method         string
		expectedStatus int
		description    string
	}{
		{
			name:           "Root path should return 418",
			path:           "/",
			method:         "GET",
			expectedStatus: http.StatusTeapot,
			description:    "Root path should return 418 I'm a teapot",
		},
		{
			name:           "Invalid path should return 418",
			path:           "/invalid-path",
			method:         "GET",
			expectedStatus: http.StatusTeapot,
			description:    "Any invalid path should return 418 I'm a teapot",
		},
		{
			name:           "Old webhook endpoint should return 418",
			path:           "/webhook",
			method:         "POST",
			expectedStatus: http.StatusTeapot,
			description:    "Old webhook endpoint should return 418 I'm a teapot",
		},
		{
			name:           "Old run-task endpoint should return 418",
			path:           "/run-task",
			method:         "POST",
			expectedStatus: http.StatusTeapot,
			description:    "Old run-task endpoint should return 418 I'm a teapot",
		},
		{
			name:           "Random endpoint should return 418",
			path:           "/some/random/path",
			method:         "GET",
			expectedStatus: http.StatusTeapot,
			description:    "Any random endpoint should return 418 I'm a teapot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			rootHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d (%s), got %d", tt.expectedStatus, http.StatusText(tt.expectedStatus), w.Code)
			}

			// Check that the response is JSON
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
			}

			// Check that the response contains teapot-related content
			var response map[string]string
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Errorf("Failed to decode JSON response: %v", err)
			}

			if tt.expectedStatus == http.StatusTeapot {
				if response["error"] != "I'm a teapot" {
					t.Errorf("Expected error message 'I'm a teapot', got '%s'", response["error"])
				}
				// Ensure no extra fields are present
				if len(response) != 1 {
					t.Errorf("Expected only 'error' field in response, got %d fields: %v", len(response), response)
				}
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode JSON response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestSecurityHeaders(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		method   string
		hasTLS   bool
		expected map[string]string
	}{
		{
			name:   "Security headers on HTTP health endpoint",
			path:   "/health",
			method: "GET",
			hasTLS: false,
			expected: map[string]string{
				"X-Content-Type-Options":   "nosniff",
				"X-Frame-Options":          "DENY",
				"X-XSS-Protection":         "1; mode=block",
				"Referrer-Policy":          "strict-origin-when-cross-origin",
				"Content-Security-Policy":  "default-src 'none'; script-src 'none'; object-src 'none'",
			},
		},
		{
			name:   "Security headers on HTTPS health endpoint with HSTS",
			path:   "/health",
			method: "GET",
			hasTLS: true,
			expected: map[string]string{
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"X-Content-Type-Options":    "nosniff",
				"X-Frame-Options":           "DENY",
				"X-XSS-Protection":          "1; mode=block",
				"Referrer-Policy":           "strict-origin-when-cross-origin",
				"Content-Security-Policy":   "default-src 'none'; script-src 'none'; object-src 'none'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			
			// Simulate TLS connection for HTTPS test
			if tt.hasTLS {
				req.TLS = &tls.ConnectionState{}
			}
			
			w := httptest.NewRecorder()

			// Apply security middleware to health handler
			handler := securityMiddleware(http.HandlerFunc(healthHandler))
			handler.ServeHTTP(w, req)

			// Check all expected headers are present
			for headerName, expectedValue := range tt.expected {
				actualValue := w.Header().Get(headerName)
				if actualValue != expectedValue {
					t.Errorf("Expected header %s: %s, got: %s", headerName, expectedValue, actualValue)
				}
			}

			// Ensure HSTS is only present for TLS connections
			hstsHeader := w.Header().Get("Strict-Transport-Security")
			if tt.hasTLS && hstsHeader == "" {
				t.Error("Expected HSTS header for TLS connection, but it was missing")
			}
			if !tt.hasTLS && hstsHeader != "" {
				t.Errorf("Expected no HSTS header for non-TLS connection, but got: %s", hstsHeader)
			}
		})
	}
}
