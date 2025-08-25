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

// VPC Flow Log constants based on AWS default format
const (
	// Flow log configuration
	VpcFlowLogsSupportedFieldCount = 14
	VpcFlowLogsSupportedVersion    = "2"

	// Telemetry names
	BytesMetricName   = "AWS.VPC.Flows.Bytes"
	PacketsMetricName = "AWS.VPC.Flows.Packets"

	// Telemetry units
	BytesUnit = "Bytes"
	CountUnit = "Count"

	// Telemetry scope
	ScopeName    = "vpc_flow_logs"
	ScopeVersion = "1.0.0"

	// Resource information
	ResourceName = "VPC Flow Logs"

	// VPC Flow Log field keys (used for field names, validation, logging, and OpenTelemetry attribute keys)
	VersionKey      = "version"
	AccountIDKey    = "account_id"
	InterfaceIDKey  = "interface_id"
	SrcAddrKey      = "src_addr"
	DstAddrKey      = "dst_addr"
	SrcPortKey      = "src_port"
	DstPortKey      = "dst_port"
	ProtocolKey     = "protocol"
	ProtocolNameKey = "protocolName"
	PacketsKey      = "packets"
	BytesKey        = "bytes"
	StartKey        = "start"
	EndKey          = "end"
	ActionKey       = "action"
	LogStatusKey    = "log_status"

	// Internal logging keys (not VPC flow log fields)
	LogGroupKey  = "log_group"
	LogStreamKey = "log_stream"
	RecordIDKey  = "record_id"
	IntervalKey  = "interval"
	JSONKey      = "json"

	// Validation constants
	MaxAttributeLength = 255
)
