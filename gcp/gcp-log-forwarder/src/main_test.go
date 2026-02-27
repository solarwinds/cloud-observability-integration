package gcp_forwarder

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetServiceName ensures logs are attributed to the correct GCP service
func TestGetServiceName(t *testing.T) {
	tests := []struct {
		name     string
		raw      map[string]any
		expected string
	}{
		{
			name:     "From LogName",
			raw:      map[string]any{"logName": "projects/vkk/logs/cloud-run-service"},
			expected: "cloud-run-service",
		},
		{
			name:     "From Resource Type",
			raw:      map[string]any{"resource": map[string]any{"type": "gcs_bucket"}},
			expected: "gcs_bucket",
		},
		{
			name:     "Unknown Fallback",
			raw:      map[string]any{},
			expected: "gcp-service-unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getServiceName(tt.raw); got != tt.expected {
				t.Errorf("getServiceName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestMapSeverity verifies the OTLP numeric severity mapping
func TestMapSeverity(t *testing.T) {
	if mapSeverity("ERROR") != 17 {
		t.Errorf("Expected ERROR to be 17, got %d", mapSeverity("ERROR"))
	}
	if mapSeverity("DEBUG") != 5 {
		t.Errorf("Expected DEBUG to be 5, got %d", mapSeverity("DEBUG"))
	}
}

// TestSendToSolarWinds mocks the HTTP call to ensure headers and compression
func TestSendToSolarWinds(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Error("Expected gzip encoding header")
		}
		if !contains(r.Header.Get("Authorization"), "Bearer") {
			t.Error("Missing Bearer token in Authorization header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	records := []map[string]any{
		{"body": "test-log"},
	}

	err := sendToSolarWinds(context.Background(), server.URL, "test-token", "test-service", records)
	if err != nil {
		t.Fatalf("sendToSolarWinds failed: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
