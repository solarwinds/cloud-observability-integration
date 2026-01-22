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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// getFlowLogFormat queries EC2 for the flow log configuration based on log group name.
func getFlowLogFormat(logGroupName string) (string, string, int, error) {
	sess := session.Must(session.NewSession())
	svc := ec2.New(sess)

	input := &ec2.DescribeFlowLogsInput{
		Filter: []*ec2.Filter{
			{
				Name:   aws.String("log-group-name"),
				Values: []*string{aws.String(logGroupName)},
			},
		},
	}

	result, err := svc.DescribeFlowLogs(input)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to describe flow logs: %w", err)
	}

	if len(result.FlowLogs) == 0 {
		return "", "", 0, fmt.Errorf("no flow logs found for log group: %s", logGroupName)
	}

	flowLog := result.FlowLogs[0] // assuming one match
	flowLogId := aws.StringValue(flowLog.FlowLogId)
	logFormat := aws.StringValue(flowLog.LogFormat)

	return logFormat, flowLogId, len(result.FlowLogs), nil
}

func parseToStruct(format string, line string, isDebugEnabled bool) (*FlowLogRecord, error) {
	formatFields := strings.Fields(format)
	logFields := strings.Fields(line)

	if len(formatFields) != len(logFields) {
		return nil, fmt.Errorf("field count mismatch: format has %d fields, line has %d", len(formatFields), len(logFields))
	}

	record := &FlowLogRecord{}
	val := reflect.ValueOf(record).Elem()
	typ := val.Type()

	for i, rawField := range formatFields {
		cleanField := strings.TrimPrefix(rawField, "${")
		cleanField = strings.TrimSuffix(cleanField, "}")

		for j := 0; j < typ.NumField(); j++ {
			field := typ.Field(j)
			jsonTag := field.Tag.Get("json")
			if jsonTag == cleanField {
				fieldVal := val.Field(j)

				// Debug: log field info only when debug is enabled
				if isDebugEnabled {
					handlerLogger.Info(fmt.Sprintf("Setting field '%s' (kind: %v, type: %v) with value '%s'",
						cleanField, fieldVal.Kind(), fieldVal.Type(), logFields[i]))
				}

				// Handle different field types
				switch fieldVal.Kind() {
				case reflect.String:
					fieldVal.SetString(logFields[i])
				case reflect.Int64:
					if intVal, err := strconv.ParseInt(logFields[i], 10, 64); err == nil {
						fieldVal.SetInt(intVal)
					} else {
						// If parsing fails, set to 0 (similar to parseInt64 function)
						fieldVal.SetInt(0)
					}
				default:
					if isDebugEnabled {
						handlerLogger.Info(fmt.Sprintf("WARNING: Unhandled field type %v for field %s", fieldVal.Kind(), cleanField))
					}
				}
				break
			}
		}
	}

	return record, nil
}
