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
)

// Note: Version validation is performed by the individual parsers (parseFlowLogRecordDefault and parseFlowLogRecordCustom)
// fieldPresence parameter: nil for default format (all fields required), or a map indicating which fields are present in custom format
func (h *Handler) validateFlowLogRecord(record *FlowLogRecord, fieldPresence FieldPresenceMap) error {
	// For custom formats, ensure all V2 default fields are present
	// Without all V2 fields, we will not send the data to OTEL
	if fieldPresence != nil {
		for _, field := range V2DefaultFieldNames {
			if !fieldPresence.HasField(field) {
				return &ValidationError{
					Field:   field,
					Message: fmt.Sprintf("Custom format must include all V2 default fields. Missing required field: '%s'", field),
				}
			}
		}
	}

	// Validate string fields that are present in the format
	// For default format (fieldPresence == nil), all these fields are required
	validateStringField := func(awsFieldName, fieldValue string) error {
		if fieldPresence == nil || fieldPresence.HasField(awsFieldName) {
			if fieldValue == "" {
				return &ValidationError{
					Field:   awsFieldName,
					Actual:  fieldValue,
					Message: fmt.Sprintf("Required field '%s' is empty or missing", awsFieldName),
				}
			}
		}
		return nil
	}

	// Validate required string fields if they are present in the format
	if err := validateStringField(ConvertKeyToAWSFieldName(VersionKey), record.Version); err != nil {
		return err
	}
	if err := validateStringField(ConvertKeyToAWSFieldName(AccountIDKey), record.AccountID); err != nil {
		return err
	}
	if err := validateStringField(ConvertKeyToAWSFieldName(InterfaceIDKey), record.InterfaceID); err != nil {
		return err
	}
	if err := validateStringField(ConvertKeyToAWSFieldName(SrcAddrKey), record.SrcAddr); err != nil {
		return err
	}
	if err := validateStringField(ConvertKeyToAWSFieldName(DstAddrKey), record.DstAddr); err != nil {
		return err
	}
	if err := validateStringField(ConvertKeyToAWSFieldName(SrcPortKey), record.SrcPort); err != nil {
		return err
	}
	if err := validateStringField(ConvertKeyToAWSFieldName(DstPortKey), record.DstPort); err != nil {
		return err
	}
	if err := validateStringField(ConvertKeyToAWSFieldName(ProtocolKey), record.Protocol); err != nil {
		return err
	}
	if err := validateStringField(ConvertKeyToAWSFieldName(ActionKey), record.Action); err != nil {
		return err
	}
	if err := validateStringField(ConvertKeyToAWSFieldName(LogStatusKey), record.LogStatus); err != nil {
		return err
	}

	// Helper function to validate numeric fields are not negative
	validateNumericField := func(awsFieldName, key string, value int64, fieldType string) error {
		if (fieldPresence == nil || fieldPresence.HasField(awsFieldName)) && value < 0 {
			return &ValidationError{
				Field:   ConvertKeyToAWSFieldName(key),
				Actual:  fmt.Sprintf("%d", value),
				Message: fmt.Sprintf("%s cannot be negative", fieldType),
			}
		}
		return nil
	}

	// Validate numeric fields are not negative (only if they are present in the format)
	if err := validateNumericField("packets", PacketsKey, record.Packets, "Packets count"); err != nil {
		return err
	}
	if err := validateNumericField("bytes", BytesKey, record.Bytes, "Bytes count"); err != nil {
		return err
	}
	if err := validateNumericField("start", StartKey, record.Start, "Start time"); err != nil {
		return err
	}
	if err := validateNumericField("end", EndKey, record.End, "End time"); err != nil {
		return err
	}

	// Validate logical time ordering (only if both fields are present)
	if (fieldPresence == nil || (fieldPresence.HasField("start") && fieldPresence.HasField("end"))) && record.Start > record.End {
		return &ValidationError{
			Field:   ConvertKeyToAWSFieldName(StartKey),
			Actual:  fmt.Sprintf("start: %d, end: %d", record.Start, record.End),
			Message: "Start time cannot be greater than end time",
		}
	}

	// Validate account ID format (only if account-id is present)
	if fieldPresence == nil || fieldPresence.HasField("account-id") {
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
	}

	// Validate action field value (only if action is present)
	if fieldPresence == nil || fieldPresence.HasField("action") {
		if record.Action != "ACCEPT" && record.Action != "REJECT" {
			return &ValidationError{
				Field:   ConvertKeyToAWSFieldName(ActionKey),
				Actual:  record.Action,
				Message: "Invalid action value (must be ACCEPT or REJECT)",
			}
		}
	}

	// Validate log status value (only if log-status is present)
	if fieldPresence == nil || fieldPresence.HasField("log-status") {
		if record.LogStatus != "OK" && record.LogStatus != "NODATA" && record.LogStatus != "SKIPDATA" {
			return &ValidationError{
				Field:   ConvertKeyToAWSFieldName(LogStatusKey),
				Actual:  record.LogStatus,
				Message: "Invalid log status (must be OK, NODATA, or SKIPDATA)",
			}
		}
	}

	return nil
}
