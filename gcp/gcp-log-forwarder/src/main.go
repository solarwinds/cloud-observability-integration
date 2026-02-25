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

var (
	httpClient    *http.Client
	storageClient *storage.Client
	bufPool       = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	gzPool        = sync.Pool{New: func() any { return gzip.NewWriter(nil) }}
)

func init() {
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	var err error
	storageClient, err = storage.NewClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to create storage client: %v", err)
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
	batches := make(map[string][]map[string]any)

	for scanner.Scan() {
		var raw map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		serviceName := getServiceName(raw)
		record := transformToOTLP(raw, data.Name)
		batches[serviceName] = append(batches[serviceName], record)

		if len(batches[serviceName]) >= 1000 {
			svc, batch := serviceName, batches[serviceName]
			g.Go(func() error { return sendToSolarWinds(ctx, swiURL, swiToken, svc, batch) })
			batches[serviceName] = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading GCS file: %w", err)
	}

	for svc, batch := range batches {
		if len(batch) > 0 {
			s, b := svc, batch
			g.Go(func() error { return sendToSolarWinds(ctx, swiURL, swiToken, s, b) })
		}
	}

	return g.Wait()
}

func transformToOTLP(raw map[string]any, fileName string) map[string]any {
	tsNano := fmt.Sprintf("%d", time.Now().UnixNano())
	for _, key := range []string{"timestamp", "time", "receiveTimestamp"} {
		if val, ok := raw[key].(string); ok && val != "" {
			if parsed, err := time.Parse(time.RFC3339, val); err == nil {
				tsNano = fmt.Sprintf("%d", parsed.UnixNano())
				break
			}
		}
	}

	severity := "INFO"
	if s, ok := raw["severity"].(string); ok {
		severity = s
	}

	return map[string]any{
		"timeUnixNano":   tsNano,
		"severityNumber": mapSeverity(severity),
		"severityText":   severity,
		"body":           recursiveMap(raw),
		"attributes": []map[string]any{
			{"key": "gcs.file_source", "value": map[string]any{"stringValue": fileName}},
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
					{"key": "telemetry.sdk.name", "value": map[string]any{"stringValue": "gcp-log-forwarder"}},
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
		return fmt.Errorf("solarwinds error: %d", resp.StatusCode)
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
	case int, int64:
		return map[string]any{"intValue": val}
	case float64:
		return map[string]any{"doubleValue": val}
	case bool:
		return map[string]any{"boolValue": val}
	default:
		return map[string]any{"stringValue": fmt.Sprintf("%v", val)}
	}
}

func getServiceName(raw map[string]any) string {
	if ln, ok := raw["logName"].(string); ok {
		parts := strings.Split(ln, "/")
		lastPart := parts[len(parts)-1]
		if lastPart != "syslog" && lastPart != "activity" {
			return lastPart
		}
	}
	if res, ok := raw["resource"].(map[string]any); ok {
		if t, ok := res["type"].(string); ok {
			return t
		}
	}
	return "gcp-service-unknown"
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
