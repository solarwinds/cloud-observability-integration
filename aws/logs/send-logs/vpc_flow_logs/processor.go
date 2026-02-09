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
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc"
)

// ProcessAndExportVpcFlowLogs processes VPC Flow Logs and exports them as metrics via gRPC
// This function encapsulates all VPC-specific processing logic, keeping main.go clean.
//
// Parameters:
//   - ctx: Context for cancellation and timeout (includes Lambda timeout)
//   - handler: VPC flow logs handler instance (should be initialized once and reused)
//   - conn: gRPC connection to OTLP endpoint
//   - owner: AWS account ID
//   - logGroup: CloudWatch log group name
//   - logStream: CloudWatch log stream name
//   - logEvents: CloudWatch log events to process
//
// Returns:
//   - successfulExports: Number of metrics successfully exported
//   - errors: Slice of errors encountered during processing/export
func ProcessAndExportVpcFlowLogs(
	ctx context.Context,
	handler *Handler,
	conn *grpc.ClientConn,
	owner, logGroup, logStream string,
	logEvents []events.CloudwatchLogsLogEvent,
) (successfulExports int, errs []error) {

	// Create metrics client for exporting to OTLP endpoint
	metricsClient := pmetricotlp.NewGRPCClient(conn)

	// Create channel for receiving processed metrics
	vpcLogChan := make(chan pmetric.Metrics)

	// Start processing VPC flow logs in a goroutine
	// The handler transforms raw log events into OpenTelemetry metrics
	go handler.TransformVpcFlowLogs(ctx, owner, logGroup, logStream, logEvents, vpcLogChan)

	// Export each metric batch as it's processed
	for processedMetric := range vpcLogChan {
		// Use parent context directly - it already has Lambda timeout
		// This allows graceful cancellation when Lambda is about to timeout
		metricRequest := pmetricotlp.NewExportRequestFromMetrics(processedMetric)
		_, err := metricsClient.Export(ctx, metricRequest)

		if err != nil {
			handlerLogger.Error("While exporting metric data: ", err.Error())
			errs = append(errs, err)
		} else {
			successfulExports++
		}
	}

	// If no metrics were successfully exported, report failure
	if successfulExports == 0 {
		errMsg := fmt.Sprintf("Failed to process any VPC flow log records from %d log events", len(logEvents))
		handlerLogger.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
	}

	return successfulExports, errs
}
