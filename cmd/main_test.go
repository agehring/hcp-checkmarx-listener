package main

import (
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
