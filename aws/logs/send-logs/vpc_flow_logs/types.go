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

// FlowLogRecord represents an AWS VPC Flow Log record (default format)
type FlowLogRecord struct {
	Version                 string `json:"version"`                    // VPC Flow Log version
	AccountID               string `json:"account-id"`                 // AWS account ID
	InterfaceID             string `json:"interface-id"`               // Network interface ID
	SrcAddr                 string `json:"srcaddr"`                    // Source IP address
	DstAddr                 string `json:"dstaddr"`                    // Destination IP address
	SrcPort                 string `json:"srcport"`                    // Source port
	DstPort                 string `json:"dstport"`                    // Destination port
	Protocol                string `json:"protocol"`                   // Protocol number
	Packets                 int64  `json:"packets"`                    // Number of packets
	Bytes                   int64  `json:"bytes"`                      // Number of bytes
	Start                   int64  `json:"start"`                      // Window start time (Unix seconds)
	End                     int64  `json:"end"`                        // Window end time (Unix seconds)
	Action                  string `json:"action"`                     // ACCEPT or REJECT
	LogStatus               string `json:"log-status"`                 // OK, NODATA, or SKIPDATA
	VpcID                   string `json:"vpc-id"`                     // VPC ID where the network interface resides
	SubnetID                string `json:"subnet-id"`                  // Subnet ID where the network interface resides
	InstanceID              string `json:"instance-id"`                // Instance ID associated with the network interface
	TcpFlags                string `json:"tcp-flags"`                  // Bitmask value for TCP flags
	Type                    string `json:"type"`                       // Type of traffic (IPv4, IPv6, EFA)
	PktSrcAddr              string `json:"pkt-srcaddr"`                // Packet-level source IP address
	PktDstAddr              string `json:"pkt-dstaddr"`                // Packet-level destination IP address
	Region                  string `json:"region"`                     // AWS region where the network interface resides
	AzID                    string `json:"az-id"`                      // Availability Zone ID where the network interface resides
	SublocationType         string `json:"sublocation-type"`           // Type of sublocation (wavelength, outpost, localzone)
	SublocationID           string `json:"sublocation-id"`             // ID of the sublocation
	PktSrcAWSService        string `json:"pkt-src-aws-service"`        // Name of AWS service that's the packet-level source
	PktDstAWSService        string `json:"pkt-dst-aws-service"`        // Name of AWS service that's the packet-level destination
	FlowDirection           string `json:"flow-direction"`             // Direction of flow relative to the interface (ingress/egress)
	TrafficPath             string `json:"traffic-path"`               // Path traffic takes from source to destination
	ECSClusterName          string `json:"ecs-cluster-name"`           // Name of the ECS cluster
	ECSClusterArn           string `json:"ecs-cluster-arn"`            // ARN of the ECS cluster
	ECSContainerInstanceID  string `json:"ecs-container-instance-id"`  // ID of the ECS container instance
	ECSContainerInstanceArn string `json:"ecs-container-instance-arn"` // ARN of the ECS container instance
	ECSServiceName          string `json:"ecs-service-name"`           // Name of the ECS service
	ECSTaskDefinitionArn    string `json:"ecs-task-definition-arn"`    // ARN of the ECS task definition
	ECSTaskID               string `json:"ecs-task-id"`                // ID of the ECS task
	ECSTaskArn              string `json:"ecs-task-arn"`               // ARN of the ECS task
	ECSContainerID          string `json:"ecs-container-id"`           // ID of the first ECS container
	ECSSecondContainerID    string `json:"ecs-second-container-id"`    // ID of the second ECS container (if applicable)
	RejectReason            string `json:"reject-reason"`              // Reason the traffic was rejected (if action is REJECT)
	ResourceID              string `json:"resource-id"`                // Resource ID (e.g., NAT gateway ID) - Available in v9+
	EncryptionStatus        string `json:"encryption-status"`          // Encryption status of the traffic - Available in v10+
}
