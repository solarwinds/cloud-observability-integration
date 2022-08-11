package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	assert "github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/model/pdata"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
)

var _= (func() interface {} {
	runningTests = true
	return nil
}())

func TestCloudTrailEventParsing(t *testing.T) {

	testCases := []struct {
		file string
		eventSource string
		instanceIdInRequest string
		instanceIdInResponse string
		region string
		err error
	} {
		{
			file: "testdata/event1.json",
			eventSource: "ec2.amazonaws.com",
			instanceIdInResponse: "i-061bf37e959383a04",
			region: "us-east-1",
		},
		{
			file: "testdata/event2.json",
			eventSource: "ec2.amazonaws.com",
			instanceIdInRequest: "i-061bf37e959383a04",
			region: "us-east-1",
		},
	}

	for _, tc:= range testCases {
		t.Run(tc.file, func (t * testing.T ) {
			// parse basic cloud trail event
			data, err := os.ReadFile(tc.file)
			if err!= nil {
				t.Fatalf("While opening %q: %q", tc.file, err)
			}
			basicEvent := cloudTrailEvent {}
			err = json.Unmarshal(data, &basicEvent)
			assert.NoError(t, err)
			assert.Equal(t, tc.eventSource, basicEvent.EventSource)

			// parse detailed cloudtrail event

			detailedEvent := ec2CloudTrailEvent {}

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
	cloudTrailGenericMessage, err := os.ReadFile("testdata/event3.json")


	testCases := [] struct {
		name string
		message string
		ok bool
		result iEc2Event
		ec2InstanceId string
		region string
	} {
		{
			name: "Plain text message detected as Default message kind",
			message: "Hello, World!",
			ok: false,
			result: nil,
		},
		{
			name: "CloudTrail EC2 event is detected and parsed",
			message: string(cloudTrailEc2Message),
			ok: true,
			result: &ec2CloudTrailEvent{
				cloudTrailEvent:   cloudTrailEvent{
					EventSource: "ec2.amazonaws.com",
					EventName:   "RunInstances",
					Region: "us-east-1",
				},
				RequestParameters: ec2InstancesSet{
					InstancesSet: ec2InstancesSetItems{
						Items: []ec2InstanceParameter {
							{},
						},
					},
				},
				ResponseElements:  ec2InstancesSet{
					InstancesSet: ec2InstancesSetItems{
						Items: []ec2InstanceParameter {
							{
								InstanceId: "i-061bf37e959383a04",
							},
						},
					},
				},
			},
			ec2InstanceId: "i-061bf37e959383a04",
			region: "us-east-1",
		},
		{
			name: "Suspected CloudTrail EC2 event message having unrecognized structure detected as Default messsage kind",
			message: "eventName ec2.amazonaws.com instancesSet",
			ok: false,
			result: nil,
		},
		{
			name: "Cluster Insights log message is detected and parsed",
			message: string(cloudInsightsLogMessage1),
			ok: true,
			result: &cloudInsightsLog{
				Ec2InstanceId: "i-test",
				Region: "us-east-1",
			},
			ec2InstanceId: "i-test",
			region: "us-east-1",
		},
		{
			name: "Cluster Insights app log message is detected and parsed",
			message: string(cloudInsightsLogMessage2),
			ok: true,
			result: &cloudInsightsAppLog{
				Kubernetes:cloudInsightsAppLogKubernetes {
					Host: "ip-127-0-0-1.us-east-2.compute.internal",
				},
				parsedInstanceId: "ip-127-0-0-1",
				parsedRegion: "us-east-2",
			},
			ec2InstanceId: "ip-127-0-0-1",
			region: "us-east-2",
		},
		{
			name: "Cluster Insights performance metrics message is detected and parsed",
			message: string(cloudInsightsLogMessage3),
			ok: true,
			result: &cloudInsightsPerformance{
				InstanceId: "i-test",
				NodeName: "ip-192-0-2-0.us-west-2.compute.internal",
				parsedRegion: "us-west-2",
			},
			ec2InstanceId: "i-test",
			region: "us-west-2",

		},
		{
			name: "CloudTrail generic message detected and parsed for region",
			message: string(cloudTrailGenericMessage),
			ok: true,
			result: &cloudTrailEvent {
					EventSource: "rds.amazonaws.com",
					EventName:   "DescribeDBInstances",
					Region: "eu-west-3",
			},
			ec2InstanceId: "",
			region: "eu-west-3",
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
	id, err := ec2Event.getInstanceId()
	assert.Equal(t, "i-061bf37e959383a04", id)
}

func TestLogEventsTransform(t *testing.T) {
	logEvents := make([] events.CloudwatchLogsLogEvent, 0)

	logEvents = append(logEvents, events.CloudwatchLogsLogEvent{
		ID:        "1",
		Timestamp: time.Now().Unix(),
		Message:   "Hello, World",
	})

	logEvents = append(logEvents,createCloudTrailCloudWatchEvent("1","testEvent", "i-12345678"))
	logEvents = append(logEvents,createCloudTrailCloudWatchEvent("1","testEvent", "another ec2 instance"))
	logEvents = append(logEvents, events.CloudwatchLogsLogEvent{
		ID:        "1",
		Timestamp: time.Now().Unix(),
		Message:   "World, hello again",
	})

	output := make(chan pdata.Logs)

	go transformLogEvents("test account", "test log group", "i-12345678", logEvents, output)

	testCases := [] struct {
		name string
		action func(t *testing.T, logs pdata.Logs)
	}   {
			{
				name : "Same host id logs are merged",
				action : func(t *testing.T, logs pdata.Logs) {
					resLogs := logs.ResourceLogs()
					assert.Equal(t, 1, resLogs.Len())
					log := resLogs.At(0)
					assert.Equal(t, 1, log.InstrumentationLibraryLogs().Len())
					instrLog := log.InstrumentationLibraryLogs().At(0)
					assert.Equal(t, 2, instrLog.Logs().Len())
				},
			},
			{
				name : "Another host id produces new logs",
				action : func(t *testing.T, logs pdata.Logs) {
					resLogs := logs.ResourceLogs()
					assert.Equal(t, 1, resLogs.Len())
					log := resLogs.At(0)
					assert.Equal(t, 1, log.InstrumentationLibraryLogs().Len())
					instrLog := log.InstrumentationLibraryLogs().At(0)
					assert.Equal(t, 1, instrLog.Logs().Len())
				},
			},
			{
				name : "Log event without host id produces new logs",
				action : func(t *testing.T, logs pdata.Logs) {
					resLogs := logs.ResourceLogs()

					assert.Equal(t, 1, resLogs.Len())
					log := resLogs.At(0)
					assert.Equal(t, 1, log.InstrumentationLibraryLogs().Len())
					instrLog := log.InstrumentationLibraryLogs().At(0)
					assert.Equal(t, 1, instrLog.Logs().Len())
					attrs := log.Resource().Attributes().AsRaw()
					hostId, _ := attrs[semconv.AttributeHostID]
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

func createCloudTrailCloudWatchEvent(logItemId, eventName, instanceId string) (evt events.CloudwatchLogsLogEvent) {
	ec2 := ec2CloudTrailEvent{
		cloudTrailEvent:   cloudTrailEvent{
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
		ResponseElements:  ec2InstancesSet{},
	}

	msg, _:= json.Marshal(ec2)
	evt = events.CloudwatchLogsLogEvent{
		ID:        logItemId,
		Timestamp: time.Now().Unix(),
		Message:   string(msg),
	}
	return
}