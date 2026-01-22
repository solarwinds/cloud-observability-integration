/* Copyright 2022 SolarWinds Worldwide, LLC. All rights reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at:
*
*	http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and limitations
* under the License.
 */

package vpc_flow_logs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"send-logs/logger"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
)

var handlerLogger = logger.NewLogger("vpc-flow-logs-handler")

// flowLogFormatGetter is a function type for retrieving flow log format
type flowLogFormatGetter func(logGroupName string) (string, string, int, error)

// Handler handles VPC Flow Log processing with debug capabilities and in-memory caching
type Handler struct {
	isDebugEnabled    bool                // Enable debug logging
	debugCounter      int                 // Counter for debug sampling
	fullDebugInterval int                 // How often to log full JSON (every Nth record)
	formatCache       *flowLogFormatCache // In-memory cache for flow log formats
	getFlowLogFormat  flowLogFormatGetter // Function to get flow log format (can be mocked in tests)
}

// NewHandler creates a new VPC flow log handler with configurable debug interval and cache TTL
func NewHandler(isDebugEnabled bool, fullDebugInterval int, cacheTTL time.Duration) *Handler {
	if fullDebugInterval <= 0 {
		fullDebugInterval = 100 // Safe default
	}
	if cacheTTL <= 0 {
		cacheTTL = 10 * time.Minute // Default cache TTL
	}

	// Create the handler with in-memory cache
	handler := &Handler{
		isDebugEnabled:    isDebugEnabled,
		debugCounter:      0,
		fullDebugInterval: fullDebugInterval,
		formatCache:       newFlowLogFormatCache(cacheTTL, isDebugEnabled),
	}

	// Set up the format getter to use the cache
	handler.getFlowLogFormat = func(logGroupName string) (string, string, int, error) {
		return handler.getFlowLogFormatCached(logGroupName)
	}

	return handler
}

// getFlowLogFormatCached retrieves flow log format with caching support
// It first checks the cache, and only makes an EC2 API call if the cache is empty or expired
func (h *Handler) getFlowLogFormatCached(logGroupName string) (string, string, int, error) {
	// Try to get from cache first
	if logFormat, flowLogId, flowLogsCount, found := h.formatCache.get(logGroupName); found {
		return logFormat, flowLogId, flowLogsCount, nil
	}

	// Cache miss or expired, query EC2
	logFormat, flowLogId, flowLogsCount, err := getFlowLogFormat(logGroupName)
	if err != nil {
		return "", "", 0, err
	}

	// Store in cache for future use
	h.formatCache.set(logGroupName, logFormat, flowLogId, flowLogsCount)

	return logFormat, flowLogId, flowLogsCount, nil
}

// isDefaultFormat checks if the format string represents AWS default format (version 2)
// Returns true if format is empty (no EC2 query was made) or matches the default format string exactly
func isDefaultFormat(format string) bool {
	return format == "" || format == VpcFlowLogsDefaultFormatString
}

// TransformVpcFlowLogs processes VPC flow log events and sends them to a metrics channel
// Context is used to handle cancellation (e.g., Lambda timeout)
func (h *Handler) TransformVpcFlowLogs(ctx context.Context, account, logGroup, logStream string, input []events.CloudwatchLogsLogEvent, output chan pmetric.Metrics) {
	defer close(output)

	flowLogsFormat := ""
	// Only query for log format if not using default version 2
	if VpcFlowLogsSupportedVersion != VpcFlowLogsDefaultVersion {
		var (
			err           error
			flowLogsCount int
			flowLogId     string
		)
		// Query EC2 for flow log format (uses injected function, can be mocked in tests)
		flowLogsFormat, flowLogId, flowLogsCount, err = h.getFlowLogFormat(logGroup)

		if err != nil {
			handlerLogger.Error("While getting flow log format: ", err.Error())
			return
		}
		if h.isDebugEnabled {
			handlerLogger.Info("LogFormat: ", flowLogsFormat)

			if flowLogsCount > 1 && h.isDebugEnabled {
				// Log a warning if multiple flow logs are found for the same log group
				handlerLogger.Info(fmt.Sprintf("WARNING: Multiple flow logs found for log group: %s. Using the first one (%s).", logGroup, flowLogId))
			}
		}
	}

	// Log parser selection ONCE before processing (avoids N log lines for N records)
	if h.isDebugEnabled {
		parserType := "DEFAULT (optimized, no reflection)"
		if !isDefaultFormat(flowLogsFormat) {
			parserType = fmt.Sprintf("CUSTOM (reflection-based). Format: %s", flowLogsFormat)
		}
		handlerLogger.Info(fmt.Sprintf("Using %s parser for %d records", parserType, len(input)))
	}

	for _, logEvent := range input {
		// Check for context cancellation (e.g., Lambda timeout)
		select {
		case <-ctx.Done():
			handlerLogger.Error("Context cancelled, stopping VPC flow log processing: ", ctx.Err().Error())
			return
		default:
			// Continue processing
		}

		var (
			record *FlowLogRecord
			err    error
		)
		// Determine which parser to use based on the actual format string from EC2
		// If format is empty (version 2, no EC2 query) or matches AWS default, use optimized default parser (no reflection)
		// Otherwise use custom parser which handles any field order via reflection
		if isDefaultFormat(flowLogsFormat) {
			record, err = h.parseFlowLogRecordDefault(logEvent.Message)
		} else {
			record, err = h.parseFlowLogRecordCustom(logEvent.Message, flowLogsFormat)
		}
		if err != nil || record == nil {
			if err != nil {
				handlerLogger.Error("Failed to parse VPC flow log record: ", err.Error())
			} else {
				handlerLogger.Error("Failed to parse VPC flow log record: record is nil")
			}
			continue
		}

		metrics := h.createMetrics(record)

		// Debug logging: Always log essential fields (cheap), full JSON only occasionally (expensive)
		if h.isDebugEnabled {
			h.debugCounter++

			// Always log essential fields - this is cheap and provides good debugging info
			handlerLogger.Info("VPC Flow Log processed",
				AccountIDKey, account,
				LogGroupKey, logGroup,
				LogStreamKey, logStream,
				VersionKey, record.Version,
				AccountIDKey, record.AccountID,
				ActionKey, record.Action,
				ProtocolKey, record.Protocol,
				ProtocolNameKey, ConvertProtocol(record.Protocol),
			)

			// Occasionally log full JSON for detailed debugging - this is expensive
			if h.debugCounter%h.fullDebugInterval == 1 { // Configurable interval for full JSON
				req := pmetricotlp.NewExportRequestFromMetrics(metrics)
				jsonMetricsRequest, _ := json.Marshal(req)
				handlerLogger.Info("Full metrics request (sample)",
					RecordIDKey, h.debugCounter,
					IntervalKey, h.fullDebugInterval,
					JSONKey, string(jsonMetricsRequest))
			}
		}

		// Send processed metrics to output channel
		output <- metrics
	}
}
