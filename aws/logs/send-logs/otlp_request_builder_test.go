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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/collector/semconv/v1.5.0"
)

func TestOltpRequestBuilder(t *testing.T) {

	rb := NewOtlpRequestBuilder()

	rb.SetCloudAccount("test account").
		SetLogGroup("test group").
		SetLogStream("test stream")

	t.Run(fmt.Sprintf("When host id is empty, %s attribute is not set", semconv.AttributeCloudPlatform), func(t *testing.T) {
		logs := rb.GetLogs()

		assert.NotNil(t, logs)
		assert.Equal(t, 1, logs.ResourceLogs().Len())
		attrs := logs.ResourceLogs().At(0).Resource().Attributes().AsRaw()

		expectedAttrs := map[string]interface{}{
			semconv.AttributeCloudProvider:     semconv.AttributeCloudProviderAWS,
			semconv.AttributeCloudAccountID:    "test account",
			semconv.AttributeAWSLogGroupNames:  "test group",
			semconv.AttributeAWSLogStreamNames: "test stream",
		}

		assert.Equal(t, expectedAttrs, attrs)
	})

	t.Run(fmt.Sprintf("When host id is not empty %s attribute is set", semconv.AttributeCloudPlatform), func(t *testing.T) {

		assert.False(t, rb.MatchHostId("test"))
		rb.SetHostId("test")
		assert.True(t, rb.MatchHostId("test"))

		logs := rb.GetLogs()

		assert.NotNil(t, logs)
		assert.Equal(t, 1, logs.ResourceLogs().Len())
		attrs := logs.ResourceLogs().At(0).Resource().Attributes().AsRaw()

		expectedAttrs := map[string]interface{}{
			semconv.AttributeHostID:            "test",
			semconv.AttributeCloudPlatform:     semconv.AttributeCloudPlatformAWSEC2,
			semconv.AttributeCloudProvider:     semconv.AttributeCloudProviderAWS,
			semconv.AttributeCloudAccountID:    "test account",
			semconv.AttributeAWSLogGroupNames:  "test group",
			semconv.AttributeAWSLogStreamNames: "test stream",
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

	rb.AddLogEntry("test entry id", time.Now().UnixMilli(), "test body", "")
	logs := rb.GetLogs()
	assert.Equal(t, 1, logs.ResourceLogs().At(0).ScopeLogs().Len())

	t.Run(fmt.Sprintf("When region is empty '%s' is not set ", semconv.AttributeCloudRegion), func(t *testing.T) {
		logEntry := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
		_, ok := logEntry.Attributes().Get(semconv.AttributeCloudRegion)
		assert.False(t, ok, fmt.Sprintf("Attribute '%s' should not be present.", semconv.AttributeCloudRegion))
	})

	region := "us-east-1"
	rb.AddLogEntry("test entry id", time.Now().UnixMilli(), "test body", region)
	logs = rb.GetLogs()

	t.Run(fmt.Sprintf("When region is provided '%s' is set to expected region ", semconv.AttributeCloudRegion), func(t *testing.T) {
		logEntry := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(1)
		regionAttr, ok := logEntry.Attributes().Get(semconv.AttributeCloudRegion)
		assert.True(t, ok, fmt.Sprintf("Attribute '%s' should be present.", semconv.AttributeCloudRegion))
		if ok {
			assert.Equal(t, region, regionAttr.Str())
		}
	})

	tcs := []struct {
		name   string
		region string
	}{
		{
			name:   "125229878893_CloudTrail_us-east-2",
			region: "us-east-2",
		},
	}
	rb = NewOtlpRequestBuilder()
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("When log stream name is '%s' parsed region equals '%s'", tc.name, tc.region), func(t *testing.T) {
			rb.SetLogStream(tc.name)
			rb.AddLogEntry("test id", time.Now().UnixMilli(), "test body", "")
			logs = rb.GetLogs()
			logIndex := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().Len() - 1
			logEntry := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(logIndex)
			regionAttr, ok := logEntry.Attributes().Get(semconv.AttributeCloudRegion)
			assert.True(t, ok, fmt.Sprintf("Attribute '%s' should be present.", semconv.AttributeCloudRegion))
			if ok {
				assert.Equal(t, tc.region, regionAttr.Str())
			}
		})
	}

	t.Run("Test regular expression", func(t *testing.T) {
		matches := detectRegionRegExp.FindStringSubmatch("125229878893_CloudTrail_us-east-2")
		assert.True(t, len(matches) > 0)
		i := detectRegionRegExp.SubexpIndex("Region")
		t.Logf("%s", matches[i])
		//t.Fail()
	})
}
