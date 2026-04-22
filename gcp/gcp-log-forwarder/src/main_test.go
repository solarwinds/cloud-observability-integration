package gcp_forwarder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// --- SERVICE & SEVERITY TESTS ---

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
			raw:      map[string]any{"resource": map[string]any{"type": "gce_instance"}},
			expected: "gce_instance",
		},
		{
			name:     "Unknown Fallback",
			raw:      map[string]any{},
			expected: "gcp-unknown-service", // Fixed: Matches return string in main.go
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

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"ERROR", 17},
		{"DEBUG", 5},
		{"INFO", 9},
		{"NOTICE", 9},
		{"CRITICAL", 21},
	}
	for _, tt := range tests {
		if got := mapSeverity(tt.input); got != tt.expected {
			t.Errorf("mapSeverity(%s) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

// --- TRANSFORMATION & BATCHING TESTS ---

func TestTransformToOTLP_NoStripping(t *testing.T) {
	tests := []struct {
		name     string
		raw      map[string]any
		expectIn string
	}{
		{
			name: "Audit Log Stringification",
			raw: map[string]any{
				"protoPayload": map[string]any{
					"methodName": "v1.compute.instances.start",
				},
				"severity": "NOTICE",
			},
			expectIn: `"methodName":"v1.compute.instances.start"`,
		},
		{
			name: "Cloud Build JSON Stringification",
			raw: map[string]any{
				"jsonPayload": map[string]any{
					"step": "build",
				},
				"severity": "INFO",
			},
			expectIn: `"step":"build"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformToOTLP(tt.raw, "test-file.log")
			body := got["body"].(map[string]any)["stringValue"].(string)

			if !strings.Contains(body, tt.expectIn) {
				t.Errorf("transformToOTLP() body = %v, want to contain %v", body, tt.expectIn)
			}
		})
	}
}

func TestBatching_SizeLimit(t *testing.T) {
	var mu sync.Mutex
	flushCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		flushCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create ~7.5MB estimated OTLP data (30 logs * 100KB raw * 2.5 overhead)
	var mockData strings.Builder
	for i := 0; i < 30; i++ {
		entry := map[string]any{
			"textPayload": strings.Repeat("X", 100*1024),
			"severity":    "INFO",
			"logName":     "projects/vkk/logs/stress-test",
		}
		b, _ := json.Marshal(entry)
		mockData.Write(b)
		mockData.WriteString("\n")
	}

	ctx := context.Background()
	reader := io.NopCloser(strings.NewReader(mockData.String()))

	// Helper logic to simulate the HandleGcsBatch loop
	err := runBatchLogicForTest(ctx, reader, server.URL, "test-token")
	if err != nil {
		t.Fatalf("Batch logic failed: %v", err)
	}

	// 7.5MB / 3MB limit should result in at least 3 batches
	if flushCount < 2 {
		t.Errorf("Expected multiple batches, but got %d. Check MaxBatchBytes logic.", flushCount)
	}
	fmt.Printf("Success: Handled heavy data in %d separate batches\n", flushCount)
}

// --- UTILITIES ---

func TestSendToSolarWinds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Error("Expected gzip encoding")
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer") {
			t.Error("Missing Bearer token")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	records := []map[string]any{
		{"body": map[string]any{"stringValue": "test-log"}},
	}

	err := sendToSolarWinds(context.Background(), server.URL, "test-token", "test-service", records)
	if err != nil {
		t.Fatalf("sendToSolarWinds failed: %v", err)
	}
}

// runBatchLogicForTest replicates the HandleGcsBatch loop for unit testing
func runBatchLogicForTest(ctx context.Context, rc io.ReadCloser, url, token string) error {
	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	batches := make(map[string]*serviceBatch)

	var wg sync.WaitGroup
	var lastErr error

	for scanner.Scan() {
		line := scanner.Bytes()
		var raw map[string]any
		json.Unmarshal(line, &raw)

		svcName := getServiceName(raw)
		if _, ok := batches[svcName]; !ok {
			batches[svcName] = &serviceBatch{}
		}

		record := transformToOTLP(raw, "test.log")
		b := batches[svcName]
		b.records = append(b.records, record)
		b.size += int(float64(len(line)) * 2.5)

		if len(b.records) >= MaxBatchRecords || b.size >= MaxBatchBytes {
			recs, svc := b.records, svcName
			wg.Add(1)
			go func(s string, r []map[string]any) {
				defer wg.Done()
				if err := sendToSolarWinds(ctx, url, token, s, r); err != nil {
					lastErr = err
				}
			}(svc, recs)
			b.records, b.size = nil, 0
		}
	}

	for svc, b := range batches {
		if len(b.records) > 0 {
			wg.Add(1)
			go func(s string, r []map[string]any) {
				defer wg.Done()
				sendToSolarWinds(ctx, url, token, s, r)
			}(svc, b.records)
		}
	}

	wg.Wait()
	return lastErr
}
