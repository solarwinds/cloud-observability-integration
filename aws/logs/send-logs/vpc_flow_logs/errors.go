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

import "fmt"

// ParseError represents a parsing error
type ParseError struct {
	Message  string
	Expected int
	Actual   int
}

func (e *ParseError) Error() string {
	return e.Message
}

// ValidationError represents a validation error for VPC Flow Log fields
type ValidationError struct {
	Field    string
	Expected string
	Actual   string
	Message  string
}

func (e *ValidationError) Error() string {
	if e.Expected != "" {
		return fmt.Sprintf("%s: expected '%s', got '%s'", e.Message, e.Expected, e.Actual)
	}
	return e.Message
}
