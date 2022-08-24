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

package main

import (
	"regexp"
	"go.opentelemetry.io/collector/model/pdata"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
)
var (
	detectHostIdRegExp = regexp.MustCompile(`^(?P<HostId>(i-|ip-)[\w\-]+)`)
	detectRegionRegExp = regexp.MustCompile(`(?P<Region>\w{2}-\w+-\d+)`)
)
type OtlpRequestBuilder interface {
	SetHostId(hostId string) (OtlpRequestBuilder)
	SetCloudAccount(account string) (OtlpRequestBuilder)
	SetLogGroup(logGroup string) (OtlpRequestBuilder)
	SetLogStream(logStream string) (OtlpRequestBuilder)
	AddLogEntry(entryId string, timestamp int64, message, region string) (OtlpRequestBuilder)
	MatchHostId(hostId string) (bool)
	HasHostId() (bool)
	GetLogs() pdata.Logs
}

type otlpRequestBuilder struct {
	logs pdata.Logs
	resLogs pdata.ResourceLogs
	instrLogsSlice pdata.InstrumentationLibraryLogsSlice
	instrLogs pdata.InstrumentationLibraryLogs
	hostId string
	parsedRegion string
	parsedHostId string
}

func NewOtlpRequestBuilder() (builder OtlpRequestBuilder){
	logs := pdata.NewLogs()
	resLogs := logs.ResourceLogs().AppendEmpty()
	resLogs.SetSchemaUrl(semconv.SchemaURL)
	instrLogsSlice := resLogs.InstrumentationLibraryLogs()
	builder = &otlpRequestBuilder{ logs :  logs, resLogs: resLogs, instrLogsSlice: instrLogsSlice}
	return
}

func (rb * otlpRequestBuilder) SetHostId(hostId string) (builder OtlpRequestBuilder) {
	rb.hostId = hostId

	attrs := rb.resLogs.Resource().Attributes()
	if rb.hostId != "" {
		attrs.UpsertString(semconv.AttributeHostID, rb.hostId)
		attrs.UpsertString(semconv.AttributeCloudPlatform, semconv.AttributeCloudPlatformAWSEC2)
	} else {
		attrs.Delete(semconv.AttributeHostID)
		attrs.Delete(semconv.AttributeCloudPlatform)
	}
	builder = rb
	return
}

func (rb * otlpRequestBuilder) SetCloudAccount(account string) (builder OtlpRequestBuilder) {
	attrs := rb.resLogs.Resource().Attributes()
	attrs.UpsertString(semconv.AttributeCloudAccountID, account)
	builder = rb
	return
}

func (rb * otlpRequestBuilder) SetLogGroup(logGroup string) (builder OtlpRequestBuilder) {
	attrs := rb.resLogs.Resource().Attributes()
	attrs.UpsertString(semconv.AttributeAWSLogGroupNames, logGroup)
	builder = rb
	return
}

func (rb * otlpRequestBuilder) SetLogStream(logStream string) (builder OtlpRequestBuilder) {
	attrs := rb.resLogs.Resource().Attributes()
	attrs.InsertString(semconv.AttributeAWSLogStreamNames, logStream)
	matches := detectHostIdRegExp.FindStringSubmatch(logStream)
	matchIndex := detectHostIdRegExp.SubexpIndex("HostId")
	if matchIndex >= 0 && matchIndex < len(matches) {
		rb.parsedHostId = matches[matchIndex]
	}

	matches = detectRegionRegExp.FindStringSubmatch(logStream)
	matchIndex = detectRegionRegExp.SubexpIndex("Region")
	if matchIndex >= 0 && matchIndex < len(matches) {
		rb.parsedRegion = matches[matchIndex]
	}

	if rb.parsedHostId != "" && !rb.HasHostId() {
		rb.SetHostId(logStream)
	}
	builder = rb
	return
}

func (rb *otlpRequestBuilder) MatchHostId(hostId string) (bool) {
	return rb.hostId == hostId
}

func (rb *otlpRequestBuilder) HasHostId() (bool) {
	return rb.hostId != ""
}

func (rb *otlpRequestBuilder) AddLogEntry(itemId string, timestamp int64, message, region string) (builder OtlpRequestBuilder) {
	if rb.instrLogsSlice.Len()== 0 {
		rb.instrLogs = rb.instrLogsSlice.AppendEmpty()
	}
	logEntry := rb.instrLogs.Logs().AppendEmpty()
	logEntry.SetName(itemId)
	logEntry.SetTimestamp(pdata.Timestamp(timestamp))
	logEntry.Body().SetStringVal(message)
	if region != "" {
		logEntry.Attributes().UpsertString(semconv.AttributeCloudRegion, region)
	} else if rb.parsedRegion != "" {
		logEntry.Attributes().UpsertString(semconv.AttributeCloudRegion, rb.parsedRegion)
	}
	builder = rb
	return
}

func (rb *otlpRequestBuilder) GetLogs() (logs pdata.Logs) {
	logs = rb.logs
	attrs := rb.resLogs.Resource().Attributes()
	attrs.InsertString(semconv.AttributeCloudProvider, semconv.AttributeCloudProviderAWS)

	return
}