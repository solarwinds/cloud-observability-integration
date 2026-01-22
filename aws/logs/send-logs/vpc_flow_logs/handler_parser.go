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
	"fmt"
	"strconv"
	"strings"

	"send-logs/logger"
)

func (h *Handler) parseFlowLogRecordDefault(message string) (*FlowLogRecord, error) {
	fields := strings.Fields(message)

	// Validate field count for AWS default format (must be exactly 14 fields)
	if len(fields) != VpcFlowLogsDefaultVersionFieldsCount {
		if h.isDebugEnabled {
			handlerLogger.Error(fmt.Sprintf("Malformed VPC flow log message: expected exactly %d fields, got %d. Message: %q", VpcFlowLogsDefaultVersionFieldsCount, len(fields), message))
		}
		return nil, &ParseError{
			Message:  "Invalid field count in VPC flow log",
			Expected: VpcFlowLogsDefaultVersionFieldsCount,
			Actual:   len(fields),
		}
	}

	// Parse according to AWS default format:
	// ${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}
	logRecord := &FlowLogRecord{
		Version:     fields[0],              // VPC Flow Log version
		AccountID:   fields[1],              // AWS account ID
		InterfaceID: fields[2],              // Network interface ID
		SrcAddr:     fields[3],              // Source IP address
		DstAddr:     fields[4],              // Destination IP address
		SrcPort:     fields[5],              // Source port
		DstPort:     fields[6],              // Destination port
		Protocol:    fields[7],              // Protocol number
		Packets:     parseInt64(fields[8]),  // Number of packets
		Bytes:       parseInt64(fields[9]),  // Number of bytes
		Start:       parseInt64(fields[10]), // Window start time
		End:         parseInt64(fields[11]), // Window end time
		Action:      fields[12],             // ACCEPT or REJECT
		LogStatus:   fields[13],             // OK, NODATA, or SKIPDATA
	}

	// Validate version for default format - require minimum version 2, allow newer versions
	// Use numeric comparison for proper version ordering (e.g., 10 > 2)
	version := parseInt64(logRecord.Version)
	minVersion := parseInt64(VpcFlowLogsDefaultVersion)
	if version < minVersion {
		return nil, &ValidationError{
			Field:   ConvertKeyToAWSFieldName(VersionKey),
			Actual:  logRecord.Version,
			Message: fmt.Sprintf("VPC Flow Log version too old (minimum: %s, got %s)", VpcFlowLogsDefaultVersion, logRecord.Version),
		}
	}

	// Log info for versions newer than tested
	supportedVersion := parseInt64(VpcFlowLogsSupportedVersion)
	if version > supportedVersion && h.isDebugEnabled {
		handlerLogger.Info(fmt.Sprintf("Processing VPC Flow Log version %s (tested up to %s). New version-specific fields may not be captured.", logRecord.Version, VpcFlowLogsSupportedVersion))
	}

	// Validate other critical fields (nil field presence means all default fields are required)
	if err := h.validateFlowLogRecord(logRecord, nil); err != nil {
		return nil, err
	}

	return logRecord, nil
}

func (h *Handler) parseFlowLogRecordCustom(message string, format string) (*FlowLogRecord, error) {
	if h.isDebugEnabled {
		handlerLogger.Info(fmt.Sprintf("Parsing VPC flow log with custom format. Format: %q, Message: %q", format, message))
	}
	logRecord, err := parseToStruct(format, message, h.isDebugEnabled)
	if err != nil {
		return nil, &ParseError{Message: fmt.Sprintf("Failed to parse VPC flow log with custom format: %v", err)}
	} else {
		if h.isDebugEnabled {
			handlerLogger.Info(fmt.Sprintf("Parsed FlowLogRecord: %+v", logRecord))
		}
	}

	fieldPresence := NewFieldPresenceMap(format)

	// Validate version for custom format - require minimum version 2, allow newer versions
	// Version is part of V2 mandatory fields, so it's always present and must be validated
	version := parseInt64(logRecord.Version)
	minVersion := parseInt64(VpcFlowLogsDefaultVersion)
	if version < minVersion {
		return nil, &ValidationError{
			Field:   ConvertKeyToAWSFieldName(VersionKey),
			Actual:  logRecord.Version,
			Message: fmt.Sprintf("VPC Flow Log version too old (minimum: %s, got %s)", VpcFlowLogsDefaultVersion, logRecord.Version),
		}
	}
	// Log info for versions newer than tested
	supportedVersion := parseInt64(VpcFlowLogsSupportedVersion)
	if version > supportedVersion && h.isDebugEnabled {
		handlerLogger.Info(fmt.Sprintf("Processing VPC Flow Log version %s (tested up to %s). New version-specific fields may not be captured.", logRecord.Version, VpcFlowLogsSupportedVersion))
	}

	// Validate other fields based on what's present in the format
	if err := h.validateFlowLogRecord(logRecord, fieldPresence); err != nil {
		return nil, err
	}

	return logRecord, nil
}

// parseInt64 parses a string to int64, returning 0 on error
func parseInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		logger.NewLogger("vpc-flow-logs-parser").Error("Error parsing integer: ", err.Error())
		return 0
	}
	return i
}
