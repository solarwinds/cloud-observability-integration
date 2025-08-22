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
	"strings"
	"unicode"
)

// ConvertKeyToAWSFieldName converts OpenTelemetry attribute key constants to AWS VPC Flow Log field names
// This converts underscores to dashes to match the actual AWS field naming convention
func ConvertKeyToAWSFieldName(key string) string {
	// Handle special cases that don't follow the simple underscore-to-dash conversion
	switch key {
	case AccountIDKey:
		return "account-id"
	case InterfaceIDKey:
		return "interface-id"
	case SrcAddrKey:
		return "srcaddr"
	case DstAddrKey:
		return "dstaddr"
	case SrcPortKey:
		return "srcport"
	case DstPortKey:
		return "dstport"
	case LogStatusKey:
		return "log-status"
	case ProtocolNameKey:
		return "protocolName" // This is not an AWS field, it's our computed field
	default:
		// For other fields, convert underscores to dashes
		return strings.ReplaceAll(key, "_", "-")
	}
}

// isValidAccountID checks if account ID is exactly 12 digits
func isValidAccountID(accountID string) bool {
	if len(accountID) != 12 {
		return false
	}
	for _, r := range accountID {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// isValidAction checks if action is ACCEPT or REJECT
func isValidAction(action string) bool {
	return action == "ACCEPT" || action == "REJECT"
}

// isValidLogStatus checks if log status is OK, NODATA, or SKIPDATA
func isValidLogStatus(logStatus string) bool {
	return logStatus == "OK" || logStatus == "NODATA" || logStatus == "SKIPDATA"
}
