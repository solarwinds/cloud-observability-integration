package gcp_forwarder

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"golang.org/x/sync/errgroup"
)

// Batching Constraints
const (
	MaxBatchRecords = 500             // Safe record count for heavy logs
	MaxBatchBytes   = 3 * 1024 * 1024 // 3MB target to stay well under the 413 limit
)

var (
	httpClient    *http.Client
	storageClient *storage.Client
	bufPool       = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	gzPool        = sync.Pool{New: func() any { return gzip.NewWriter(nil) }}
)

// serviceBatch tracks state for multiple services found in a single GCS file
type serviceBatch struct {
	records []map[string]any
	size    int
}

func init() {
	// Initialize HTTP Client with production timeouts
	httpClient = &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	// Initialize Storage Client once at startup
	var err error
	storageClient, err = storage.NewClient(context.Background())
	if err != nil {
		log.Fatalf("failed to create storage client: %v", err)
	}

	functions.CloudEvent("HandleGcsBatch", HandleGcsBatch)
}

func HandleGcsBatch(ctx context.Context, e event.Event) error {
	var data struct {
		Bucket string `json:"bucket"`
		Name   string `json:"name"`
	}
	if err := e.DataAs(&data); err != nil {
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	rc, err := storageClient.Bucket(data.Bucket).Object(data.Name).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to open GCS object: %w", err)
	}
	defer rc.Close()

	swiURL := os.Getenv("SWI_OTEL_ENDPOINT")
	swiToken := os.Getenv("SWI_API_KEY")

	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	g, ctx := errgroup.WithContext(ctx)
	batches := make(map[string]*serviceBatch)

	flush := func(svc string, b *serviceBatch) {
		currentRecords, currentSvc := b.records, svc
		g.Go(func() error {
			return sendToSolarWinds(ctx, swiURL, swiToken, currentSvc, currentRecords)
		})
		b.records = nil
		b.size = 0
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		svcName := getServiceName(raw)
		if _, ok := batches[svcName]; !ok {
			batches[svcName] = &serviceBatch{}
		}

		record := transformToOTLP(raw, data.Name)

		b := batches[svcName]
		b.records = append(b.records, record)
		// Estimate bloat: OTLP JSON wrapping adds significant overhead
		b.size += int(float64(len(line)) * 2.5)

		if len(b.records) >= MaxBatchRecords || b.size >= MaxBatchBytes {
			flush(svcName, b)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading GCS file: %w", err)
	}

	// Final flush for remaining logs
	for svc, b := range batches {
		if len(b.records) > 0 {
			flush(svc, b)
		}
	}

	return g.Wait()
}

// transformToOTLP maps GCP fields to the OTLP schema
func transformToOTLP(raw map[string]any, fileName string) map[string]any {
	var bodyText string

	// 1. Audit Logs (protoPayload)
	if pp, ok := raw["protoPayload"].(map[string]any); ok {
		// Serialize the entire protoPayload directly to the body
		jb, _ := json.Marshal(pp)
		bodyText = string(jb)
	}

	// 2. Structured Logs (jsonPayload)
	if bodyText == "" {
		if jp, ok := raw["jsonPayload"].(map[string]any); ok {
			if msg, ok := jp["message"].(string); ok {
				bodyText = msg
			} else {
				// Serialize the whole jsonPayload
				jb, _ := json.Marshal(jp)
				bodyText = string(jb)
			}
		}
	}

	// 3. Simple Text Logs
	if bodyText == "" {
		if tp, ok := raw["textPayload"].(string); ok {
			bodyText = tp
		}
	}

	// Final Fallback
	if bodyText == "" {
		bodyText = "GCP Log Entry"
	}

	return map[string]any{
		"timeUnixNano":   fmt.Sprintf("%d", time.Now().UnixNano()),
		"severityNumber": mapSeverity(fmt.Sprintf("%v", raw["severity"])),
		"severityText":   fmt.Sprintf("%v", raw["severity"]),
		"body":           map[string]any{"stringValue": bodyText},
		"attributes": []map[string]any{
			{"key": "gcs.file_source", "value": map[string]any{"stringValue": fileName}},
			{"key": "log.metadata", "value": recursiveMap(raw)},
		},
	}
}

func sendToSolarWinds(ctx context.Context, url, token, serviceName string, records []map[string]any) error {
	payload := map[string]any{
		"resourceLogs": []map[string]any{{
			"resource": map[string]any{
				"attributes": []map[string]any{
					{"key": "service.name", "value": map[string]any{"stringValue": serviceName}},
					{"key": "cloud.provider", "value": map[string]any{"stringValue": "gcp"}},
				},
			},
			"scopeLogs": []map[string]any{{"logRecords": records}},
		}},
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	gz := gzPool.Get().(*gzip.Writer)
	gz.Reset(buf)
	if err := json.NewEncoder(gz).Encode(payload); err != nil {
		return err
	}
	gz.Close()
	gzPool.Put(gz)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, buf)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SolarWinds error: %d for service %s", resp.StatusCode, serviceName)
	}
	return nil
}

func recursiveMap(v any) map[string]any {
	switch val := v.(type) {
	case map[string]any:
		kvList := make([]map[string]any, 0, len(val))
		for k, child := range val {
			kvList = append(kvList, map[string]any{"key": k, "value": recursiveMap(child)})
		}
		return map[string]any{"kvlistValue": map[string]any{"values": kvList}}
	case []any:
		arr := make([]map[string]any, 0, len(val))
		for _, item := range val {
			arr = append(arr, recursiveMap(item))
		}
		return map[string]any{"arrayValue": map[string]any{"values": arr}}
	default:
		return map[string]any{"stringValue": fmt.Sprintf("%v", val)}
	}
}

func getServiceName(raw map[string]any) string {
	if ln, ok := raw["logName"].(string); ok {
		parts := strings.Split(ln, "/")
		last := parts[len(parts)-1]
		if !strings.Contains(last, "syslog") && !strings.Contains(last, "activity") {
			return last
		}
	}
	if res, ok := raw["resource"].(map[string]any); ok {
		if t, ok := res["type"].(string); ok {
			return t
		}
	}
	return "gcp-unknown-service"
}

func mapSeverity(s string) int {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return 5
	case "INFO", "NOTICE":
		return 9
	case "WARNING":
		return 13
	case "ERROR":
		return 17
	case "CRITICAL", "ALERT", "EMERGENCY":
		return 21
	default:
		return 9
	}
}
