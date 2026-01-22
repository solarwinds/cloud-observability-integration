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

import "strings"

// FieldPresenceMap tracks which fields are present in a VPC flow log format
// This is used to validate only the fields that are actually included in custom formats
type FieldPresenceMap map[string]bool

func NewFieldPresenceMap(format string) FieldPresenceMap {
	if format == "" || format == VpcFlowLogsDefaultFormatString {
		// For default format, return nil to indicate all default fields should be validated
		return nil
	}

	presence := make(FieldPresenceMap)
	fields := strings.Fields(format)

	for _, rawField := range fields {
		// Remove ${} delimiters
		cleanField := strings.TrimPrefix(rawField, "${")
		cleanField = strings.TrimSuffix(cleanField, "}")
		presence[cleanField] = true
	}

	return presence
}

// For default format (nil map), all default fields are considered present
func (f FieldPresenceMap) HasField(awsFieldName string) bool {
	if f == nil {
		return isDefaultFormatField(awsFieldName)
	}
	return f[awsFieldName]
}

func isDefaultFormatField(awsFieldName string) bool {
	return defaultFieldsMap[awsFieldName]
}
