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
//   - handler.go: Main handler struct and constructor for processing VPC Flow Logs
//   - processor.go: High-level processor function for processing and exporting metrics via gRPC
//   - protocol.go: Protocol number to name conversion utilities
//   - errors.go: Custom error types for parsing and validation
//   - utils.go: Utility functions for sanitization and validation
//   - memory_cache.go: In-memory cache for VPC Flow Log format strings
//   - flow_logs_parser.go: Parser for retrieving VPC Flow Log format from AWS EC2
//
// Example usage from main.go:
//
//	// Initialize handler once (typically in init())
//	handler := vpc_flow_logs.NewHandler(true, 100, 10*time.Minute)
//
//	// Process and export VPC Flow Logs
//	successCount, errors := vpc_flow_logs.ProcessAndExportVpcFlowLogs(
//	    ctx, handler, grpcConn, accountID, logGroup, logStream, logEvents,
//	)
package vpc_flow_logs
