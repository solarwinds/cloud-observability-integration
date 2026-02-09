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
	VpcFlowLogsDefaultVersionFieldsCount = 14
	VpcFlowLogsSupportedVersion          = "10"
	VpcFlowLogsDefaultVersion            = "2" // AWS default format version
	// AWS default format string as returned by EC2 DescribeFlowLogs API
	VpcFlowLogsDefaultFormatString = "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}"

	// Telemetry names
	BytesMetricName   = "AWS.VPC.Flows.Bytes"
	PacketsMetricName = "AWS.VPC.Flows.Packets"

	// Telemetry units
	BytesUnit = "Bytes"
	CountUnit = "Count"

	// Resource information
	ResourceName = "VPC Flow Logs"

	// VPC Flow Log field keys (used for field names, validation, logging, and OpenTelemetry attribute keys)
	VersionKey                 = "version"
	AccountIDKey               = "account_id"
	InterfaceIDKey             = "interface_id"
	SrcAddrKey                 = "src_addr"
	DstAddrKey                 = "dst_addr"
	SrcPortKey                 = "src_port"
	DstPortKey                 = "dst_port"
	ProtocolKey                = "protocol"
	ProtocolNameKey            = "protocolName"
	PacketsKey                 = "packets"
	BytesKey                   = "bytes"
	StartKey                   = "start"
	EndKey                     = "end"
	ActionKey                  = "action"
	LogStatusKey               = "log_status"
	VpcIDKey                   = "vpc_id"
	SubnetIDKey                = "subnet_id"
	InstanceIDKey              = "instance_id"
	TcpFlagsKey                = "tcp_flags"
	TypeKey                    = "type"
	PktSrcAddrKey              = "pkt_srcaddr"
	PktDstAddrKey              = "pkt_dstaddr"
	RegionKey                  = "region"
	AzIDKey                    = "az_id"
	SublocationTypeKey         = "sublocation_type"
	SublocationIDKey           = "sublocation_id"
	PktSrcAWSServiceKey        = "pkt_src_aws_service"
	PktDstAWSServiceKey        = "pkt_dst_aws_service"
	FlowDirectionKey           = "flow_direction"
	TrafficPathKey             = "traffic_path"
	ECSClusterNameKey          = "ecs_cluster_name"
	ECSClusterArnKey           = "ecs_cluster_arn"
	ECSContainerInstanceIDKey  = "ecs_container_instance_id"
	ECSContainerInstanceArnKey = "ecs_container_instance_arn"
	ECSServiceNameKey          = "ecs_service_name"
	ECSTaskDefinitionArnKey    = "ecs_task_definition_arn"
	ECSTaskIDKey               = "ecs_task_id"
	ECSTaskArnKey              = "ecs_task_arn"
	ECSContainerIDKey          = "ecs_container_id"
	ECSSecondContainerIDKey    = "ecs_second_container_id"
	RejectReasonKey            = "reject_reason"
	ResourceIDKey              = "resource_id"
	EncryptionStatusKey        = "encryption_status"

	// Internal logging keys (not VPC flow log fields)
	LogGroupKey  = "log_group"
	LogStreamKey = "log_stream"
	RecordIDKey  = "record_id"
	IntervalKey  = "interval"
	JSONKey      = "json"

	// Validation constants
	MaxAttributeLength = 255
)

// V2DefaultFieldNames contains all fields in the AWS VPC Flow Logs V2 default format.
// These fields are mandatory for custom formats to ensure data completeness.
// Reference: https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs.html#flow-logs-default
var V2DefaultFieldNames = []string{
	"version", "account-id", "interface-id", "srcaddr", "dstaddr",
	"srcport", "dstport", "protocol", "packets", "bytes",
	"start", "end", "action", "log-status",
}

// defaultFieldsMap is a package-level map for efficient field lookup.
// Used by isDefaultFormatField to avoid repeated map allocations.
var defaultFieldsMap map[string]bool

func init() {
	defaultFieldsMap = make(map[string]bool, len(V2DefaultFieldNames))
	for _, field := range V2DefaultFieldNames {
		defaultFieldsMap[field] = true
	}
}
