package main

import (
	"strings"

	"go.opentelemetry.io/collector/model/pdata"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
)

type OtlpRequestBuilder interface {
	SetHostId(hostId string) (OtlpRequestBuilder)
	SetCloudAccount(account string) (OtlpRequestBuilder)
	SetLogGroup(logGroup string) (OtlpRequestBuilder)
	SetLogStream(logStream string) (OtlpRequestBuilder)
	AddLogEntry(entryId string, timestamp int64, message string) (OtlpRequestBuilder)
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
	if strings.HasPrefix( logStream, "i-") && !rb.HasHostId() {
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

func (rb *otlpRequestBuilder) AddLogEntry(itemId string, timestamp int64, message string) (builder OtlpRequestBuilder) {
	if rb.instrLogsSlice.Len()== 0 {
		rb.instrLogs = rb.instrLogsSlice.AppendEmpty()
	}
	logEntry := rb.instrLogs.Logs().AppendEmpty()
	logEntry.SetName(itemId)
	logEntry.SetTimestamp(pdata.Timestamp(timestamp))
	logEntry.Body().SetStringVal(message)
	builder = rb
	return
}

func (rb *otlpRequestBuilder) GetLogs() (logs pdata.Logs) {
	logs = rb.logs
	attrs := rb.resLogs.Resource().Attributes()
	attrs.InsertString(semconv.AttributeCloudProvider, semconv.AttributeCloudProviderAWS)

	return
}