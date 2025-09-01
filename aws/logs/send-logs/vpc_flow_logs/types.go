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
	Version         string `json:"version"`      // Field 0: VPC Flow Log version
	AccountID       string `json:"account-id"`   // Field 1: AWS account ID
	InterfaceID     string `json:"interface-id"` // Field 2: Network interface ID
	SourceAddr      string `json:"srcaddr"`      // Field 3: Source IP address
	DestinationAddr string `json:"dstaddr"`      // Field 4: Destination IP address
	SourcePort      string `json:"srcport"`      // Field 5: Source port
	DestinationPort string `json:"dstport"`      // Field 6: Destination port
	Protocol        string `json:"protocol"`     // Field 7: Protocol number
	Packets         int64  `json:"packets"`      // Field 8: Number of packets
	Bytes           int64  `json:"bytes"`        // Field 9: Number of bytes
	Start           int64  `json:"start"`        // Field 10: Window start time (Unix seconds)
	End             int64  `json:"end"`          // Field 11: Window end time (Unix seconds)
	Action          string `json:"action"`       // Field 12: ACCEPT or REJECT
	LogStatus       string `json:"log-status"`   // Field 13: OK, NODATA, or SKIPDATA
}
