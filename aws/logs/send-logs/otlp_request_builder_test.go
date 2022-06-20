package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
)

func TestOltpRequestBuilder(t *testing.T) {

	rb := NewOtlpRequestBuilder()

	rb.SetCloudAccount("test account").
		SetLogGroup("test group").
		SetLogStream("test stream")



	t.Run(fmt.Sprintf( "When host id is empty, %s attribute is not set", semconv.AttributeCloudPlatform), func (t *testing.T) {
		logs := rb.GetLogs()

		assert.NotNil(t, logs)
		assert.Equal(t, 1, logs.ResourceLogs().Len())
		attrs := logs.ResourceLogs().At(0).Resource().Attributes().AsRaw()

		expectedAttrs := map [string] interface {} {
			semconv.AttributeCloudProvider : semconv.AttributeCloudProviderAWS,
			semconv.AttributeCloudAccountID : "test account",
			semconv.AttributeAWSLogGroupNames : "test group",
			semconv.AttributeAWSLogStreamNames : "test stream",
		}

		assert.Equal(t, expectedAttrs, attrs)
	})

	t.Run(fmt.Sprintf( "When host id is not empty %s attribute is set", semconv.AttributeCloudPlatform), func (t *testing.T) {

		assert.False(t, rb.MatchHostId("test"))
		rb.SetHostId("test")
		assert.True(t, rb.MatchHostId("test"))

		logs := rb.GetLogs()

		assert.NotNil(t, logs)
		assert.Equal(t, 1, logs.ResourceLogs().Len())
		attrs := logs.ResourceLogs().At(0).Resource().Attributes().AsRaw()

		expectedAttrs := map [string] interface {} {
			semconv.AttributeHostID : "test",
			semconv.AttributeCloudPlatform: semconv.AttributeCloudPlatformAWSEC2,
			semconv.AttributeCloudProvider : semconv.AttributeCloudProviderAWS,
			semconv.AttributeCloudAccountID : "test account",
			semconv.AttributeAWSLogGroupNames : "test group",
			semconv.AttributeAWSLogStreamNames : "test stream",
		}

		assert.Equal(t, expectedAttrs, attrs)
	})

	t.Run("When log stream starts with 'i-' host id and platform is set", func(t *testing.T) {
		rb.SetHostId("")
		rb.SetLogStream("i-12345-test")
		assert.True(t, rb.MatchHostId("i-12345-test"))
		attrs := rb.GetLogs().ResourceLogs().At(0).Resource().Attributes().AsRaw()
		assert.Contains(t, attrs, semconv.AttributeCloudPlatform)
		assert.Contains(t, attrs, semconv.AttributeHostID)
	})


	rb.AddLogEntry("test entry id", time.Now().UnixMilli(), "test body")
	logs := rb.GetLogs()
	assert.Equal(t, 1, logs.ResourceLogs().At(0).InstrumentationLibraryLogs().Len())
}