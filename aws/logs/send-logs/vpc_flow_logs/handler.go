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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"send-logs/logger"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	semconv "go.opentelemetry.io/collector/semconv/v1.27.0"
)

var handlerLogger = logger.NewLogger("vpc-flow-logs-handler")

// Handler handles VPC Flow Log processing with debug capabilities
type Handler struct {
	isDebugEnabled    bool // Enable debug logging
	debugCounter      int  // Counter for debug sampling
	fullDebugInterval int  // How often to log full JSON (every Nth record)
}

// NewHandler creates a new VPC flow log handler with configurable debug interval
func NewHandler(isDebugEnabled bool, fullDebugInterval int) *Handler {
	if fullDebugInterval <= 0 {
		fullDebugInterval = 100 // Safe default
	}
	return &Handler{
		isDebugEnabled:    isDebugEnabled,
		debugCounter:      0,
		fullDebugInterval: fullDebugInterval,
	}
}

// TransformVpcFlowLogs processes VPC flow log events and sends them to a metrics channel
func (h *Handler) TransformVpcFlowLogs(account, logGroup, logStream string, input []events.CloudwatchLogsLogEvent, output chan pmetric.Metrics) {
	defer close(output)

	for _, logEvent := range input {
		record, err := h.parseFlowLogRecord(logEvent.Message)
		if err != nil {
			handlerLogger.Error("Failed to parse VPC flow log record: ", err.Error())
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

// parseFlowLogRecord parses an AWS VPC Flow Log message (default format) into a FlowLogRecord
func (h *Handler) parseFlowLogRecord(message string) (*FlowLogRecord, error) {
	fields := strings.Fields(message)

	// Validate field count for AWS default format (must be exactly 14 fields)
	if len(fields) != VpcFlowLogsSupportedFieldCount {
		if h.isDebugEnabled {
			handlerLogger.Error(fmt.Sprintf("Malformed VPC flow log message: expected exactly %d fields, got %d. Message: %q", VpcFlowLogsSupportedFieldCount, len(fields), message))
		}
		errorMessage := "Invalid field count in VPC flow log"
		if len(fields) < VpcFlowLogsSupportedFieldCount {
			errorMessage = "Insufficient fields in VPC flow log"
		} else {
			errorMessage = "Too many fields in VPC flow log"
		}
		return nil, &ParseError{
			Message:  errorMessage,
			Expected: VpcFlowLogsSupportedFieldCount,
			Actual:   len(fields),
		}
	}

	// Parse according to AWS default format:
	// ${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}
	logRecord := &FlowLogRecord{
		Version:         fields[0],              // VPC Flow Log version
		AccountID:       fields[1],              // AWS account ID
		InterfaceID:     fields[2],              // Network interface ID
		SourceAddr:      fields[3],              // Source IP address
		DestinationAddr: fields[4],              // Destination IP address
		SourcePort:      fields[5],              // Source port
		DestinationPort: fields[6],              // Destination port
		Protocol:        fields[7],              // Protocol number
		Packets:         parseInt64(fields[8]),  // Number of packets
		Bytes:           parseInt64(fields[9]),  // Number of bytes
		Start:           parseInt64(fields[10]), // Window start time
		End:             parseInt64(fields[11]), // Window end time
		Action:          fields[12],             // ACCEPT or REJECT
		LogStatus:       fields[13],             // OK, NODATA, or SKIPDATA
	}

	// Validate critical fields
	if err := h.validateFlowLogRecord(logRecord); err != nil {
		return nil, err
	}

	return logRecord, nil
}

// validateFlowLogRecord validates critical fields in the VPC Flow Log record
func (h *Handler) validateFlowLogRecord(record *FlowLogRecord) error {
	// Validate version (should be "2" for default format)
	if record.Version != VpcFlowLogsSupportedVersion {
		return &ValidationError{
			Field:    ConvertKeyToAWSFieldName(VersionKey),
			Expected: VpcFlowLogsSupportedVersion,
			Actual:   record.Version,
			Message:  "Unsupported VPC Flow Log version",
		}
	}

	// Validate account ID (should be 12 digits)
	if len(record.AccountID) != 12 {
		return &ValidationError{
			Field:   ConvertKeyToAWSFieldName(AccountIDKey),
			Actual:  record.AccountID,
			Message: "Invalid AWS account ID format (expected 12 digits)",
		}
	}

	// Validate that account ID contains only digits
	for _, r := range record.AccountID {
		if r < '0' || r > '9' {
			return &ValidationError{
				Field:   ConvertKeyToAWSFieldName(AccountIDKey),
				Actual:  record.AccountID,
				Message: "Invalid AWS account ID format (must contain only digits)",
			}
		}
	}

	// Validate action field
	if record.Action != "ACCEPT" && record.Action != "REJECT" {
		return &ValidationError{
			Field:   ConvertKeyToAWSFieldName(ActionKey),
			Actual:  record.Action,
			Message: "Invalid action value (must be ACCEPT or REJECT)",
		}
	}

	// Validate log status
	if record.LogStatus != "OK" && record.LogStatus != "NODATA" && record.LogStatus != "SKIPDATA" {
		return &ValidationError{
			Field:   ConvertKeyToAWSFieldName(LogStatusKey),
			Actual:  record.LogStatus,
			Message: "Invalid log status (must be OK, NODATA, or SKIPDATA)",
		}
	}

	return nil
}

// createMetrics creates OpenTelemetry metrics from a VPC flow log record
func (h *Handler) createMetrics(logRecord *FlowLogRecord) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.SetSchemaUrl(semconv.SchemaURL)
	rm.Resource().Attributes().PutStr("Name", ResourceName)

	ilms := rm.ScopeMetrics().AppendEmpty()
	ilms.SetSchemaUrl(semconv.SchemaURL)
	ilms.Scope().SetName(ScopeName)
	ilms.Scope().SetVersion(ScopeVersion)

	// Byte Metric
	byteMetric := ilms.Metrics().AppendEmpty()
	byteMetric.SetName(BytesMetricName)
	byteMetric.SetDescription("Bytes transferred in VPC flow logs")
	byteMetric.SetUnit(BytesUnit)
	byteMetric.SetEmptyGauge()

	byteDP := byteMetric.Gauge().DataPoints().AppendEmpty()

	byteDP.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(logRecord.Start, 0)))
	byteDP.SetIntValue(logRecord.Bytes)
	h.insertAttributes(&byteDP, logRecord)

	// Packet Metric
	packetMetric := ilms.Metrics().AppendEmpty()
	packetMetric.SetName(PacketsMetricName)
	packetMetric.SetDescription("Packets transferred in VPC flow logs")
	packetMetric.SetUnit(CountUnit)
	packetMetric.SetEmptyGauge()

	packetDP := packetMetric.Gauge().DataPoints().AppendEmpty()
	packetDP.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(logRecord.Start, 0)))
	packetDP.SetIntValue(logRecord.Packets)
	h.insertAttributes(&packetDP, logRecord)

	return metrics
}

// insertAttributes adds VPC flow log attributes to a metric data point (AWS default format)
func (h *Handler) insertAttributes(dataPoint *pmetric.NumberDataPoint, logRecord *FlowLogRecord) {
	// Define a map of string attributes for AWS default format only
	stringAttributes := map[string]string{
		VersionKey:      sanitizeAttributeValue(logRecord.Version, MaxAttributeLength),
		AccountIDKey:    sanitizeAttributeValue(logRecord.AccountID, MaxAttributeLength),
		InterfaceIDKey:  sanitizeAttributeValue(logRecord.InterfaceID, MaxAttributeLength),
		SrcAddrKey:      sanitizeAttributeValue(logRecord.SourceAddr, MaxAttributeLength),
		DstAddrKey:      sanitizeAttributeValue(logRecord.DestinationAddr, MaxAttributeLength),
		SrcPortKey:      sanitizeAttributeValue(logRecord.SourcePort, MaxAttributeLength),
		DstPortKey:      sanitizeAttributeValue(logRecord.DestinationPort, MaxAttributeLength),
		ProtocolKey:     sanitizeAttributeValue(logRecord.Protocol, MaxAttributeLength),
		ProtocolNameKey: sanitizeAttributeValue(ConvertProtocol(logRecord.Protocol), MaxAttributeLength),
		ActionKey:       sanitizeAttributeValue(logRecord.Action, MaxAttributeLength),
		LogStatusKey:    sanitizeAttributeValue(logRecord.LogStatus, MaxAttributeLength),
	}

	// Insert string attributes
	for key, value := range stringAttributes {
		dataPoint.Attributes().PutStr(key, value)
	}

	// Insert integer attributes for AWS default format
	dataPoint.Attributes().PutInt(StartKey, logRecord.Start)
	dataPoint.Attributes().PutInt(EndKey, logRecord.End)
}

// parseInt64 parses a string to int64, returning 0 on error
func parseInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		handlerLogger.Error("Error parsing integer: ", err.Error())
		return 0
	}
	return i
}

// sanitizeAttributeValue sanitizes a string value before inserting it as an attribute.
// It removes control characters, trims long values, and ensures the value is clean and valid for OpenTelemetry.
func sanitizeAttributeValue(value string, maxLength int) string {
	// Step 1: Remove any control characters (e.g., non-printable ASCII characters).
	var sanitized []rune
	for _, r := range value {
		if unicode.IsPrint(r) {
			sanitized = append(sanitized, r)
		}
	}

	// Step 2: Trim the string to the maximum allowed length (if necessary).
	sanitizedStr := string(sanitized)
	if len(sanitizedStr) > maxLength {
		sanitizedStr = sanitizedStr[:maxLength]
	}

	// Return the sanitized value
	return sanitizedStr
}
