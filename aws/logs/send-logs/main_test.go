package main

import (
	"encoding/json"
	"os"
	"testing"
	assert "github.com/stretchr/testify/assert"
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
		err error
	} {
		{
			file: "testdata/event1.json",
			eventSource: "ec2.amazonaws.com",
			instanceIdInResponse: "i-061bf37e959383a04",
		},
		{
			file: "testdata/event2.json",
			eventSource: "ec2.amazonaws.com",
			instanceIdInRequest: "i-061bf37e959383a04",
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
		})
	}

	t.Log("TestCloudTrailEventParsing")
}