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
	"time"
	"unicode"

	"send-logs/scope"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	semconv "go.opentelemetry.io/collector/semconv/v1.27.0"
)

// createMetrics creates OpenTelemetry metrics from a VPC flow log record
func (h *Handler) createMetrics(logRecord *FlowLogRecord) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.SetSchemaUrl(semconv.SchemaURL)
	rm.Resource().Attributes().PutStr("Name", ResourceName)

	ilms := rm.ScopeMetrics().AppendEmpty()
	ilms.SetSchemaUrl(semconv.SchemaURL)
	scope.SetInstrumentationScope(ilms.Scope())

	// Byte Metric
	byteMetric := ilms.Metrics().AppendEmpty()
	byteMetric.SetName(BytesMetricName)
	byteMetric.SetDescription("Bytes transferred in VPC flow logs")
	byteMetric.SetUnit(BytesUnit)
	byteMetric.SetEmptyGauge()

	byteDP := byteMetric.Gauge().DataPoints().AppendEmpty()

	byteDP.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(logRecord.Start, 0)))
	byteDP.SetIntValue(logRecord.Bytes)
	h.insertAttributes(&byteDP, logRecord)

	// Packet Metric
	packetMetric := ilms.Metrics().AppendEmpty()
	packetMetric.SetName(PacketsMetricName)
	packetMetric.SetDescription("Packets transferred in VPC flow logs")
	packetMetric.SetUnit(CountUnit)
	packetMetric.SetEmptyGauge()

	packetDP := packetMetric.Gauge().DataPoints().AppendEmpty()
	packetDP.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(logRecord.Start, 0)))
	packetDP.SetIntValue(logRecord.Packets)
	h.insertAttributes(&packetDP, logRecord)

	return metrics
}

// insertAttributes adds VPC flow log attributes to a metric data point
// Only adds attributes for fields that have non-empty values to handle custom formats gracefully
func (h *Handler) insertAttributes(dataPoint *pmetric.NumberDataPoint, logRecord *FlowLogRecord) {
	// Helper function to add string attribute only if value is not empty
	addStringAttr := func(key, value string) {
		if sanitized := sanitizeAttributeValue(value, MaxAttributeLength); sanitized != "" {
			dataPoint.Attributes().PutStr(key, sanitized)
		}
	}

	// Add string attributes only if they have non-empty values
	addStringAttr(VersionKey, logRecord.Version)
	addStringAttr(AccountIDKey, logRecord.AccountID)
	addStringAttr(InterfaceIDKey, logRecord.InterfaceID)
	addStringAttr(SrcAddrKey, logRecord.SrcAddr)
	addStringAttr(DstAddrKey, logRecord.DstAddr)
	addStringAttr(SrcPortKey, logRecord.SrcPort)
	addStringAttr(DstPortKey, logRecord.DstPort)
	addStringAttr(ProtocolKey, logRecord.Protocol)
	addStringAttr(ProtocolNameKey, ConvertProtocol(logRecord.Protocol))
	addStringAttr(ActionKey, logRecord.Action)
	addStringAttr(LogStatusKey, logRecord.LogStatus)
	addStringAttr(VpcIDKey, logRecord.VpcID)
	addStringAttr(SubnetIDKey, logRecord.SubnetID)
	addStringAttr(InstanceIDKey, logRecord.InstanceID)
	addStringAttr(TcpFlagsKey, logRecord.TcpFlags)
	addStringAttr(TypeKey, logRecord.Type)
	addStringAttr(PktSrcAddrKey, logRecord.PktSrcAddr)
	addStringAttr(PktDstAddrKey, logRecord.PktDstAddr)
	addStringAttr(RegionKey, logRecord.Region)
	addStringAttr(AzIDKey, logRecord.AzID)
	// Additional fields for version 3 and later
	addStringAttr(SublocationTypeKey, logRecord.SublocationType)
	addStringAttr(SublocationIDKey, logRecord.SublocationID)
	addStringAttr(PktSrcAWSServiceKey, logRecord.PktSrcAWSService)
	addStringAttr(PktDstAWSServiceKey, logRecord.PktDstAWSService)
	addStringAttr(FlowDirectionKey, logRecord.FlowDirection)
	addStringAttr(TrafficPathKey, logRecord.TrafficPath)
	addStringAttr(ECSClusterNameKey, logRecord.ECSClusterName)
	addStringAttr(ECSClusterArnKey, logRecord.ECSClusterArn)
	addStringAttr(ECSContainerInstanceIDKey, logRecord.ECSContainerInstanceID)
	addStringAttr(ECSContainerInstanceArnKey, logRecord.ECSContainerInstanceArn)
	addStringAttr(ECSServiceNameKey, logRecord.ECSServiceName)
	addStringAttr(ECSTaskDefinitionArnKey, logRecord.ECSTaskDefinitionArn)
	addStringAttr(ECSTaskIDKey, logRecord.ECSTaskID)
	addStringAttr(ECSTaskArnKey, logRecord.ECSTaskArn)
	addStringAttr(ECSContainerIDKey, logRecord.ECSContainerID)
	addStringAttr(ECSSecondContainerIDKey, logRecord.ECSSecondContainerID)
	addStringAttr(RejectReasonKey, logRecord.RejectReason)
	addStringAttr(ResourceIDKey, logRecord.ResourceID)
	addStringAttr(EncryptionStatusKey, logRecord.EncryptionStatus)

	// Insert integer attributes (always add start and end as they are required for telemetry)
	dataPoint.Attributes().PutInt(StartKey, logRecord.Start)
	dataPoint.Attributes().PutInt(EndKey, logRecord.End)
}

func sanitizeAttributeValue(value string, maxLength int) string {
	// Step 1: Remove any control characters (e.g., non-printable ASCII characters).
	var sanitized []rune
	for _, r := range value {
		if unicode.IsPrint(r) {
			sanitized = append(sanitized, r)
		}
	}

	// Step 2: Trim the string to the maximum allowed length (if necessary).
	sanitizedStr := string(sanitized)
	if len(sanitizedStr) > maxLength {
		sanitizedStr = sanitizedStr[:maxLength]
	}

	// Return the sanitized value
	return sanitizedStr
}
