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

// Package vpc_flow_logs provides comprehensive VPC Flow Log processing capabilities
// for AWS Lambda functions, including parsing, validation, and OpenTelemetry metrics generation.
//
// This package is organized into the following files:
//   - constants.go: All constants for field names, attribute keys, and validation values
//   - types.go: Data structures for VPC Flow Log records
//   - handler.go: Main handler struct and constructor
//   - protocol.go: Protocol number to name conversion utilities
//   - errors.go: Custom error types for parsing and validation
//   - utils.go: Utility functions for sanitization and validation
//   - processing.go: Core processing logic for parsing and metrics generation
//
// Example usage:
//
//	handler := vpc_flow_logs.NewHandler(true, 100)
//	handler.TransformVpcFlowLogs("123456789012", "vpc-logs", "stream1", events, output)
package vpc_flow_logs
