package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	assert "github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	semconv "go.opentelemetry.io/collector/semconv/v1.5.0"
)

var _ = (func() interface{} {
	runningTests = true
	return nil
}())

func TestCloudTrailEventParsing(t *testing.T) {

	testCases := []struct {
		file                 string
		eventSource          string
		instanceIdInRequest  string
		instanceIdInResponse string
		region               string
		err                  error
		k8sPodName           string
		k8sNamespaceName     string
		k8sPodID             string
		k8sHost              string
		k8sContainerName     string
		k8sDockerID          string
		k8sContainerImage    string
	}{
		{
			file:                 "testdata/event1.json",
			eventSource:          "ec2.amazonaws.com",
			instanceIdInResponse: "i-061bf37e959383a04",
			region:               "us-east-1",
		},
		{
			file:                "testdata/event2.json",
			eventSource:         "ec2.amazonaws.com",
			instanceIdInRequest: "i-061bf37e959383a04",
			region:              "us-east-1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			// parse basic cloud trail event
			data, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatalf("While opening %q: %q", tc.file, err)
			}
			basicEvent := cloudTrailEvent{}
			err = json.Unmarshal(data, &basicEvent)
			assert.NoError(t, err)
			assert.Equal(t, tc.eventSource, basicEvent.EventSource)

			// parse detailed cloudtrail event

			detailedEvent := ec2CloudTrailEvent{}

			err = json.Unmarshal(data, &detailedEvent)
			assert.NoError(t, err)
			if tc.instanceIdInRequest != "" {

				instanceId, err := extractEC2InstanceId(&detailedEvent)
				assert.NoError(t, err)
				assert.Equal(t, tc.instanceIdInRequest, instanceId)
			}

			if tc.instanceIdInResponse != "" {
				instanceId, err := extractEC2InstanceId(&detailedEvent)
				assert.NoError(t, err)
				assert.Equal(t, tc.instanceIdInResponse, instanceId)
			}

			if tc.k8sPodName != "" {
				instanceId, err := extractEC2InstanceId(&detailedEvent)
				assert.NoError(t, err)
				assert.Equal(t, tc.instanceIdInResponse, instanceId)
			}

			assert.Equal(t, tc.region, detailedEvent.getRegion())
		})
	}

	t.Log("TestCloudTrailEventParsing")
}

func TestMessageKindDetection(t *testing.T) {
	cloudTrailEc2Message, err := os.ReadFile("testdata/event1.json")
	assert.Nil(t, err)
	cloudInsightsLogMessage1, err := os.ReadFile("testdata/cloud_insights_log.json")
	assert.Nil(t, err)
	cloudInsightsLogMessage2, err := os.ReadFile("testdata/cloud_insights_app_log.json")
	assert.Nil(t, err)
	cloudInsightsLogMessage3, err := os.ReadFile("testdata/cloud_insights_perf.json")
	assert.Nil(t, err)
	cloudInsightsLogMessage4, err := os.ReadFile("testdata/cloud_insights_app_fargate_log.json")
	assert.Nil(t, err)
	cloudTrailGenericMessage, _ := os.ReadFile("testdata/event3.json")

	testCases := []struct {
		name          string
		message       string
		ok            bool
		result        iEc2Event
		ec2InstanceId string
		region        string
	}{
		{
			name:    "Plain text message detected as Default message kind",
			message: "Hello, World!",
			ok:      false,
			result:  nil,
		},
		{
			name:    "CloudTrail EC2 event is detected and parsed",
			message: string(cloudTrailEc2Message),
			ok:      true,
			result: &ec2CloudTrailEvent{
				cloudTrailEvent: cloudTrailEvent{
					EventSource: "ec2.amazonaws.com",
					EventName:   "RunInstances",
					Region:      "us-east-1",
				},
				RequestParameters: ec2InstancesSet{
					InstancesSet: ec2InstancesSetItems{
						Items: []ec2InstanceParameter{
							{},
						},
					},
				},
				ResponseElements: ec2InstancesSet{
					InstancesSet: ec2InstancesSetItems{
						Items: []ec2InstanceParameter{
							{
								InstanceId: "i-061bf37e959383a04",
							},
						},
					},
				},
			},
			ec2InstanceId: "i-061bf37e959383a04",
			region:        "us-east-1",
		},
		{
			name:    "Suspected CloudTrail EC2 event message having unrecognized structure detected as Default messsage kind",
			message: "eventName ec2.amazonaws.com instancesSet",
			ok:      false,
			result:  nil,
		},
		{
			name:    "Cluster Insights log message is detected and parsed",
			message: string(cloudInsightsLogMessage1),
			ok:      true,
			result: &cloudInsightsLog{
				Ec2InstanceId: "i-test",
				Region:        "us-east-1",
			},
			ec2InstanceId: "i-test",
			region:        "us-east-1",
		},
		{
			name:    "Cluster Insights app fargate log message is detected and parsed",
			message: string(cloudInsightsLogMessage4),
			ok:      true,
			result: &cloudInsightsAppLog{
				Kubernetes: cloudInsightsAppLogKubernetes{
					Host:           "fargate-ip-192-168-149-22.us-east-2.compute.internal",
					PodName:        "php-app-7657497f69-vfvtf",
					NamespaceName:  "faragate-namespace",
					PodID:          "d9ecc709-b396-4e8a-a041-ebb49d98a5c6",
					ContainerName:  "php-app",
					DockerID:       "5f08ea472f14acc17caf0e32ab56030fbb950f6960c41ae1d40f63c34c842a7a",
					ContainerImage: "php:8.0-apache-bullseye",
					Labels: map[string]string{
						"app":                               "php-app",
						"eks.amazonaws.com/fargate-profile": "fargate-test-cluster-profile",
						"pod-template-hash":                 "7657497f69",
					},
					Annotations: map[string]string{
						"CapacityProvisioned": "0.25vCPU 0.5GB",
						"Logging":             "LoggingEnabled",
					},
				},
				ClusterUID:       "someClusterUid",
				LogType:          "container",
				Stream:           "stderr",
				Logtag:           "F",
				Log:              "AH00558: apache2: Could not reliably determine the server's fully qualified domain name, using 192.168.149.22. Set the 'ServerName' directive globally to suppress this message",
				parsedInstanceId: "",
				parsedRegion:     "us-east-2",
			},
			ec2InstanceId: "",
			region:        "us-east-2",
		},
		{
			name:    "Cluster Insights app log message is detected and parsed",
			message: string(cloudInsightsLogMessage2),
			ok:      true,
			result: &cloudInsightsAppLog{
				Kubernetes: cloudInsightsAppLogKubernetes{
					Host:           "ip-127-0-0-1.us-east-2.compute.internal",
					PodName:        "test",
					NamespaceName:  "amazon-cloudwatch",
					PodID:          "test",
					ContainerName:  "test",
					DockerID:       "test",
					ContainerImage: "amazon/test:2.10.0",
					Labels:         nil,
					Annotations:    nil,
				},
				ClusterUID:       "",
				LogType:          "",
				Stream:           "stderr",
				Logtag:           "",
				Log:              "[2022/06/07 10:34:46] [ info] [output:cloudwatch_logs:cloudwatch_logs.0] Sent 57 events to CloudWatch\n",
				parsedInstanceId: "ip-127-0-0-1",
				parsedRegion:     "us-east-2",
			},
			ec2InstanceId: "ip-127-0-0-1",
			region:        "us-east-2",
		},
		{
			name:    "Cluster Insights performance metrics message is detected and parsed",
			message: string(cloudInsightsLogMessage3),
			ok:      true,
			result: &cloudInsightsPerformance{
				InstanceId:   "i-test",
				NodeName:     "ip-192-0-2-0.us-west-2.compute.internal",
				parsedRegion: "us-west-2",
			},
			ec2InstanceId: "i-test",
			region:        "us-west-2",
		},
		{
			name:    "CloudTrail generic message detected and parsed for region",
			message: string(cloudTrailGenericMessage),
			ok:      true,
			result: &cloudTrailEvent{
				EventSource: "rds.amazonaws.com",
				EventName:   "DescribeDBInstances",
				Region:      "eu-west-3",
			},
			ec2InstanceId: "",
			region:        "eu-west-3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ok, result := parseMessage(tc.message)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.result, result)
			if ok {
				ec2InstanceId, _ := result.getInstanceId()
				assert.Equal(t, tc.ec2InstanceId, ec2InstanceId)
				region := result.getRegion()
				assert.Equal(t, tc.region, region)
			}
		})
	}
}

func TestMessageParsing(t *testing.T) {
	cloudTrailEc2Message, err := os.ReadFile("testdata/event1.json")
	assert.Nil(t, err)
	ok, ec2Event := parseMessage(string(cloudTrailEc2Message))

	assert.True(t, ok)
	id, _ := ec2Event.getInstanceId()
	assert.Equal(t, "i-061bf37e959383a04", id)
}

func TestLogEventsTransform(t *testing.T) {
	logEvents := make([]events.CloudwatchLogsLogEvent, 0)

	logEvents = append(logEvents, events.CloudwatchLogsLogEvent{
		ID:        "1",
		Timestamp: time.Now().Unix(),
		Message:   "Hello, World",
	})

	logEvents = append(logEvents, createCloudTrailCloudWatchEvent("1", "testEvent", "i-12345678"))
	logEvents = append(logEvents, createCloudTrailCloudWatchEvent("1", "testEvent", "another ec2 instance"))
	logEvents = append(logEvents, events.CloudwatchLogsLogEvent{
		ID:        "1",
		Timestamp: time.Now().Unix(),
		Message:   "World, hello again",
	})

	output := make(chan plog.Logs)

	go transformLogEvents("test account", "test log group", "i-12345678", logEvents, output)

	testCases := []struct {
		name   string
		action func(t *testing.T, logs plog.Logs)
	}{
		{
			name: "Same host id logs are merged",
			action: func(t *testing.T, logs plog.Logs) {
				resLogs := logs.ResourceLogs()
				assert.Equal(t, 1, resLogs.Len())
				log := resLogs.At(0)
				assert.Equal(t, 1, log.ScopeLogs().Len())
				instrLog := log.ScopeLogs().At(0)
				assert.Equal(t, 2, instrLog.LogRecords().Len())
			},
		},
		{
			name: "Another host id produces new logs",
			action: func(t *testing.T, logs plog.Logs) {
				resLogs := logs.ResourceLogs()
				assert.Equal(t, 1, resLogs.Len())
				log := resLogs.At(0)
				assert.Equal(t, 1, log.ScopeLogs().Len())
				instrLog := log.ScopeLogs().At(0)
				assert.Equal(t, 1, instrLog.LogRecords().Len())
			},
		},
		{
			name: "Log event without host id produces new logs",
			action: func(t *testing.T, logs plog.Logs) {
				resLogs := logs.ResourceLogs()

				assert.Equal(t, 1, resLogs.Len())
				log := resLogs.At(0)
				assert.Equal(t, 1, log.ScopeLogs().Len())
				instrLog := log.ScopeLogs().At(0)
				assert.Equal(t, 1, instrLog.LogRecords().Len())
				attrs := log.Resource().Attributes().AsRaw()
				hostId := attrs[semconv.AttributeHostID]
				assert.Equal(t, "i-12345678", hostId)
			},
		},
	}
	testCaseIndex := 0
	for log := range output {
		assert.Less(t, testCaseIndex, len(testCases))
		tc := testCases[testCaseIndex]
		t.Run(tc.name, func(t *testing.T) {
			tc.action(t, log)
		})

		testCaseIndex += 1
	}
}

func TestTestJsonPath(t *testing.T) {
	// Sample JSON event for testing
	jsonEvent := map[string]interface{}{
		"eventSource": "aws:s3",
		"requestParameters": map[string]interface{}{
			"instancesSet": map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": "i-123456"},
					map[string]interface{}{"id": "i-789012"},
				},
			},
		},
	}

	tests := []struct {
		path     string
		value    string
		expected bool
	}{
		{"eventSource", "aws:s3", true},
		{"eventSource", "aws:ec2", false},
		{"requestParameters.instancesSet", "", true},
		{"requestParameters.instancesSet.items", "", true},
		{"requestParameters.nonexistent", "", false},
		{"nonexistent", "", false},
		{"requestParameters.instancesSet.items.id", "", false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			got := testJsonPath(jsonEvent, test.path, test.value)
			if got != test.expected {
				t.Errorf("For path %s with value %s, expected %v but got %v", test.path, test.value, test.expected, got)
			}
		})
	}
}

func TestLogEventsTransformForFargateTwoDifferentContainers(t *testing.T) {
	sampleMessage, err := os.ReadFile("testdata/cloud_insights_app_fargate_log.json")
	assert.Nil(t, err)
	sampleMessage2, err := os.ReadFile("testdata/cloud_insights_app_fargate_log2.json")
	assert.Nil(t, err)

	logEvent := events.CloudwatchLogsLogEvent{
		ID:        "eventId1",
		Timestamp: 1612550597000,
		Message:   string(sampleMessage),
	}
	logEvent2 := events.CloudwatchLogsLogEvent{
		ID:        "eventId2",
		Timestamp: 1612550597000,
		Message:   string(sampleMessage2),
	}
	inputLogEvents := []events.CloudwatchLogsLogEvent{logEvent, logEvent2}

	logsChan := make(chan plog.Logs)
	go transformLogEvents("123456789012", "/aws/lambda/MyFunction", "2022/02/06/[$LATEST]abcd1234", inputLogEvents, logsChan)
	transformedLogs := <-logsChan

	assert.NotNil(t, transformedLogs)
	assert.Equal(t, 1, transformedLogs.ResourceLogs().Len(), "Expected exactly one ResourceLogs entry")

	logRecord := transformedLogs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attributes := logRecord.Attributes()
	resourceAttributes := transformedLogs.ResourceLogs().At(0).Resource().Attributes()

	assert.Equal(t, "AH00558: apache2: Could not reliably determine the server's fully qualified domain name, using 192.168.149.22. Set the 'ServerName' directive globally to suppress this message", logRecord.Body().Str())

	// Validate Kubernetes attributes
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.pod.name", "php-app-7657497f69-vfvtf")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.namespace.name", "faragate-namespace")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.container.image.name", "php:8.0-apache-bullseye")
	assertLogRecordHasAttribute(t, resourceAttributes, "container.id", "5f08ea472f14acc17caf0e32ab56030fbb950f6960c41ae1d40f63c34c842a7a")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.pod.uid", "d9ecc709-b396-4e8a-a041-ebb49d98a5c6")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.container.name", "php-app")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.node.name", "fargate-ip-192-168-149-22.us-east-2.compute.internal")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.pod.labels.app", "php-app")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.pod.labels.eks.amazonaws.com/fargate-profile", "fargate-test-cluster-profile")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.pod.labels.pod-template-hash", "7657497f69")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.pod.annotations.CapacityProvisioned", "0.25vCPU 0.5GB")
	assertLogRecordHasAttribute(t, resourceAttributes, "k8s.pod.annotations.Logging", "LoggingEnabled")
	assertLogRecordHasAttribute(t, resourceAttributes, "host.name", "php-app-7657497f69-vfvtf")
	assertLogRecordHasAttribute(t, resourceAttributes, "service.name", "php-app")
	assertLogRecordHasAttribute(t, attributes, "sw.k8s.log.type", "container")
	assertLogRecordDoNotHaveAttribute(t, attributes, "syslog.facility")
	assertLogRecordDoNotHaveAttribute(t, attributes, "syslog.version")
	assertLogRecordDoNotHaveAttribute(t, attributes, "syslog.procid")
	assertLogRecordDoNotHaveAttribute(t, attributes, "syslog.msgid")

	transformedLogs2 := <-logsChan

	assert.NotNil(t, transformedLogs2)
	assert.Equal(t, 1, transformedLogs2.ResourceLogs().Len(), "Expected exactly one ResourceLogs entry")

	logRecord2 := transformedLogs2.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attributes2 := logRecord2.Attributes()
	resourceAttributes2 := transformedLogs2.ResourceLogs().At(0).Resource().Attributes()

	assert.Equal(t, "AH00558: apache2: Could not reliably determine the server's fully qualified domain name, using 192.168.149.22. Set the 'ServerName' directive globally to suppress this message", logRecord2.Body().Str())

	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.pod.name", "php-app-7657497f69-1234")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.namespace.name", "faragate-namespace")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.container.image.name", "php:8.0-apache-bullseye")
	assertLogRecordHasAttribute(t, resourceAttributes2, "container.id", "5f08ea472f14acc17caf0e32ab56030fbb950f6960c41ae1d40f63c34c841234")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.pod.uid", "d9ecc709-b396-4e8a-a041-ebb49d981234")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.container.name", "php-app")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.node.name", "fargate-ip-192-168-149-22.us-east-2.compute.internal")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.pod.labels.app", "php-app")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.pod.labels.eks.amazonaws.com/fargate-profile", "fargate-test-cluster-profile")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.pod.labels.pod-template-hash", "7657497f69")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.pod.annotations.CapacityProvisioned", "0.25vCPU 0.5GB")
	assertLogRecordHasAttribute(t, resourceAttributes2, "k8s.pod.annotations.Logging", "LoggingEnabled")
	assertLogRecordHasAttribute(t, resourceAttributes2, "host.name", "php-app-7657497f69-1234")
	assertLogRecordHasAttribute(t, resourceAttributes2, "service.name", "php-app")
	assertLogRecordHasAttribute(t, attributes2, "sw.k8s.log.type", "container")
	assertLogRecordDoNotHaveAttribute(t, attributes2, "syslog.facility")
	assertLogRecordDoNotHaveAttribute(t, attributes2, "syslog.version")
	assertLogRecordDoNotHaveAttribute(t, attributes2, "syslog.procid")
	assertLogRecordDoNotHaveAttribute(t, attributes2, "syslog.msgid")
}

func assertLogRecordHasAttribute(t *testing.T, attributes pcommon.Map, key string, expectedValue string) {
	val, ok := attributes.Get(key)
	assert.True(t, ok)
	assert.Equal(t, expectedValue, val.Str())
}

func assertLogRecordDoNotHaveAttribute(t *testing.T, attributes pcommon.Map, key string) {
	_, ok := attributes.Get(key)
	assert.False(t, ok)
}

func createCloudTrailCloudWatchEvent(logItemId, eventName, instanceId string) (evt events.CloudwatchLogsLogEvent) {
	ec2 := ec2CloudTrailEvent{
		cloudTrailEvent: cloudTrailEvent{
			EventSource: "ec2.amazonaws.com",
			EventName:   eventName,
		},
		RequestParameters: ec2InstancesSet{
			InstancesSet: ec2InstancesSetItems{
				Items: []ec2InstanceParameter{
					{
						InstanceId: instanceId,
					},
				},
			},
		},
		ResponseElements: ec2InstancesSet{},
	}

	msg, _ := json.Marshal(ec2)
	evt = events.CloudwatchLogsLogEvent{
		ID:        logItemId,
		Timestamp: time.Now().Unix(),
		Message:   string(msg),
	}
	return
}
