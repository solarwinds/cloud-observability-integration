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
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"regexp"
	"send-logs/logger"
	"send-logs/vpc_flow_logs"
	"strconv"
	"strings"

	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// enum for supported event types
const (
	fargateEvent = "fargate"
	ec2Event     = "ec2"
)

const (
	awsLambdaFunctionNameVar = "AWS_LAMBDA_FUNCTION_NAME"
	awsLambdaInitTypeVar     = "AWS_LAMBDA_INITIALIZATION_TYPE"
	awsRegionVar             = "AWS_REGION"
	awsFunctionVersion       = "AWS_LAMBDA_FUNCTION_VERSION"
	otlpEndpointVar          = "OTLP_ENDPOINT"
	apiTokenVar              = "API_TOKEN"
	useEncryptionVar         = "USE_ENCRYPTION"
	timestampMultiplier      = 1000000 // AWS Logs timestamp is in millisends since Jan 1 , 1970, OTEL Collector timestamp is in nanoseconds
	vpcLogGroupName          = "VPC_LOG_GROUP_NAME"
	logLevel                 = "LOG_LEVEL"
	vpcDebugInterval         = "VPC_DEBUG_INTERVAL" // How often to log full JSON (every Nth record)
)

var (
	runningTests                       = false
	functionName                string = os.Getenv(awsLambdaFunctionNameVar)
	_, executingInAWS                  = os.LookupEnv(awsLambdaInitTypeVar)
	lambdaRegion                string = os.Getenv(awsRegionVar)
	lambdaVersion               string = os.Getenv(awsFunctionVersion)
	useEncryption                      = executingInAWS && strings.EqualFold(os.Getenv(useEncryptionVar), "yes")
	endpoint                    string = os.Getenv(otlpEndpointVar) // encrypted when AWS_EXECUTION_ENV contains 'AWS_Lambda_'
	apiToken                    string = os.Getenv(apiTokenVar)     // encrypted when AWS_EXECUTION_ENV contains 'AWS_Lambda_'
	appLogger                          = logger.NewLogger("send-logs")
	kmsClient                   *kms.KMS
	detectInstanceNameAndRegion        = regexp.MustCompile(`(?P<Fargate>(fargate-))?(?P<Instance>(i-|ip-)[\w\-]+)\.(?P<Region>[\w\-]+)\.`)
	instanceParamIndex                 = detectInstanceNameAndRegion.SubexpIndex("Instance")
	regionParamIndex                   = detectInstanceNameAndRegion.SubexpIndex("Region")
	fargateParamIndex                  = detectInstanceNameAndRegion.SubexpIndex("Fargate")
	vpcLogGrpName               string = os.Getenv(vpcLogGroupName)
	isDebugEnabled              bool   = strings.EqualFold(os.Getenv(logLevel), "DEBUG")
	vpcDebugIntervalValue       int    = getVpcDebugInterval()
)

type cloudTrailEvent struct {
	EventSource string `json:"eventSource"`
	EventName   string `json:"eventName"`
	Region      string `json:"awsRegion"`
}

type ec2InstanceParameter struct {
	InstanceId string `json:"instanceId"`
}
type ec2InstancesSetItems struct {
	Items []ec2InstanceParameter `json:"items"`
}

type ec2InstancesSet struct {
	InstancesSet ec2InstancesSetItems `json:"instancesSet"`
}
type ec2CloudTrailEvent struct {
	cloudTrailEvent
	RequestParameters ec2InstancesSet `json:"requestParameters"`
	ResponseElements  ec2InstancesSet `json:"responseElements"`
}

type cloudInsightsLog struct {
	Ec2InstanceId string `json:"ec2_instance_id"`
	Region        string `json:"az"`
}

type cloudInsightsAppLogKubernetes struct {
	PodName        string            `json:"pod_name"`
	NamespaceName  string            `json:"namespace_name"`
	PodID          string            `json:"pod_id"`
	Labels         map[string]string `json:"labels"`
	Annotations    map[string]string `json:"annotations"`
	ContainerName  string            `json:"container_name"`
	DockerID       string            `json:"docker_id"`
	ContainerImage string            `json:"container_image"`
	Host           string            `json:"host"`
}

type cloudInsightsAppLog struct {
	Kubernetes       cloudInsightsAppLogKubernetes `json:"kubernetes"`
	ClusterUID       string                        `json:"sw.k8s.cluster.uid"`
	LogType          string                        `json:"sw.k8s.log.type"`
	ManifestVersion  string                        `json:"sw.k8s.agent.manifest.version"`
	Stream           string                        `json:"stream"`
	Logtag           string                        `json:"logtag"`
	Log              string                        `json:"log"`
	parsedInstanceId string
	parsedRegion     string
}

type cloudInsightsPerformance struct {
	InstanceId   string `json:"InstanceId"`
	NodeName     string `json:"NodeName"`
	parsedRegion string
}

type iEc2Event interface {
	getInstanceId() (string, error)
	getRegion() string
	getEventType() string
}

func init() {

	if runningTests {
		return
	}

	if endpoint == "" || apiToken == "" {
		appLogger.Fatal(fmt.Sprintf("Function execution parameters are not configured. Please set and encrypt %s and %s environmet variables", otlpEndpointVar, apiTokenVar))
	}

	if !useEncryption {
		// not depolyed to AWS or USE_ENCRYPTION != yes, skip decryption
		appLogger.Info("Skipping parameter decryption.")
		return
	}

	kmsClient = kms.New(session.New())
	endpoint = decodeString(endpoint)
	apiToken = decodeString(apiToken)
}

func decodeString(encrypted string) string {
	decodedBytes, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		appLogger.Fatal(err)
	}
	input := &kms.DecryptInput{
		CiphertextBlob: decodedBytes,
		EncryptionContext: aws.StringMap(map[string]string{
			"LambdaFunctionName": functionName,
		}),
	}
	response, err := kmsClient.Decrypt(input)
	if err != nil {
		appLogger.Fatal(err)
	}

	return string(response.Plaintext[:])
}

// getVpcDebugInterval parses the VPC_DEBUG_INTERVAL environment variable
// Returns a safe default of 100 if not set or invalid
func getVpcDebugInterval() int {
	intervalStr := os.Getenv(vpcDebugInterval)
	if intervalStr == "" {
		return 100 // Default: log full JSON every 100th record
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		appLogger.Error(fmt.Sprintf("VPC_DEBUG_INTERVAL: unable to parse '%s' as number, using default 100", intervalStr))
		return 100
	}

	// Check boundary conditions with specific error messages
	if interval < 1 {
		appLogger.Error(fmt.Sprintf("VPC_DEBUG_INTERVAL can't be less than 1, got %d, using default 100", interval))
		return 100
	}

	// Set reasonable upper bounds
	if interval > 10000 {
		appLogger.Error(fmt.Sprintf("VPC_DEBUG_INTERVAL too large (max 10000), got %d, capping at 10000", interval))
		return 10000
	}

	return interval
}

func extractEC2InstanceId(ec2Event *ec2CloudTrailEvent) (instanceId string, err error) {
	if len(ec2Event.RequestParameters.InstancesSet.Items) > 0 {
		instanceId = ec2Event.RequestParameters.InstancesSet.Items[0].InstanceId
		if instanceId != "" {
			return
		}
	}

	if len(ec2Event.ResponseElements.InstancesSet.Items) > 0 {
		instanceId = ec2Event.ResponseElements.InstancesSet.Items[0].InstanceId
		if instanceId != "" {
			return
		}
	}
	err = errors.New("Instance Id is not present")
	return
}

func handleEvent(ctx context.Context, event events.CloudwatchLogsEvent) (r string, err error) {
	r = "failure"
	datareq, err := event.AWSLogs.Parse()
	if err != nil {
		appLogger.Error("While parsing Cloudwatch Log event: ", err.Error())
		return r, err
	}

	dialOption := grpc.WithInsecure()

	if executingInAWS {
		config := &tls.Config{}
		dialOption = grpc.WithTransportCredentials(credentials.NewTLS(config))
	}

	conn, err := grpc.Dial(endpoint, dialOption)

	if err != nil {
		appLogger.Error("While connecting to otlp/gRPC endpoint: ", err.Error())
		return r, err
	}

	defer conn.Close()

	errs := make([]error, 0)
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+apiToken)

	// Check if this is a VPC log group
	if datareq.LogGroup == vpcLogGrpName {
		// Process VPC flow logs as metrics
		metricsClient := pmetricotlp.NewGRPCClient(conn)
		vpcLogChan := make(chan pmetric.Metrics)

		// process VPC flow logs using the handler with channel pattern
		vpcHandler := vpc_flow_logs.NewHandler(isDebugEnabled, vpcDebugIntervalValue)
		go vpcHandler.TransformVpcFlowLogs(datareq.Owner, datareq.LogGroup, datareq.LogStream, datareq.LogEvents, vpcLogChan)

		for processedMetric := range vpcLogChan {
			metricRequest := pmetricotlp.NewExportRequestFromMetrics(processedMetric)
			_, err := metricsClient.Export(ctx, metricRequest)
			if err != nil {
				appLogger.Error("While exporting metric data: ", err.Error())
				errs = append(errs, err)
			}
		}
	} else {
		// Process regular logs
		logsClient := plogotlp.NewGRPCClient(conn)
		logsChan := make(chan plog.Logs)

		go transformLogEvents(datareq.Owner, datareq.LogGroup, datareq.LogStream, datareq.LogEvents, logsChan)

		for logsData := range logsChan {
			logRequest := plogotlp.NewExportRequestFromLogs(logsData)
			_, err = logsClient.Export(ctx, logRequest)
			if err != nil {
				appLogger.Error("While exporting log data: ", err.Error())
				errs = append(errs, err)
			}
		}
	}
	if len(errs) == 0 {
		r = "success"
	} else {
		err = errs[len(errs)-1]
	}
	appLogger.Info("Function execution result: ", r)

	return r, err
}

func transformLogEvents(account, logGroup, logStream string, input []events.CloudwatchLogsLogEvent, output chan plog.Logs) {
	defer close(output)
	reqBuilder := NewOtlpRequestBuilder().
		SetCloudAccount(account).
		SetLogGroup(logGroup).
		SetLogStream(logStream)

	for _, item := range input {

		// normalize timestamp to be accepted by OTEL
		timestamp := item.Timestamp * timestampMultiplier

		ok, ec2Event := parseMessage(item.Message)

		if ok {
			instanceId, err := ec2Event.getInstanceId()
			if err == nil {
				if !reqBuilder.HasHostId() {
					reqBuilder.SetHostId(instanceId)
				} else if !reqBuilder.MatchHostId(instanceId) {
					output <- reqBuilder.GetLogs()
					reqBuilder = NewOtlpRequestBuilder().
						SetCloudAccount(account).
						SetLogGroup(logGroup).
						SetLogStream(logStream).
						SetHostId(instanceId)
				}
			}

			if ec2Event.getEventType() == fargateEvent {
				k8sFargateLog := ec2Event.(*cloudInsightsAppLog)

				if !reqBuilder.HasContainerName() {
					setKubernetesInfo(reqBuilder, k8sFargateLog)
				} else if !reqBuilder.MatchContainerName(k8sFargateLog.ClusterUID, k8sFargateLog.Kubernetes.NamespaceName, k8sFargateLog.Kubernetes.PodName, k8sFargateLog.Kubernetes.ContainerName) {
					// new container, send logs for previous container
					output <- reqBuilder.GetLogs()
					reqBuilder = setKubernetesInfo(
						NewOtlpRequestBuilder().
							SetCloudAccount(account).
							SetLogGroup(logGroup).
							SetLogStream(logStream),
						k8sFargateLog)
				}

				reqBuilder.AddLogEntry(item.ID, timestamp, k8sFargateLog.Log, ec2Event.getRegion(), map[string]interface{}{
					"sw.k8s.log.type": k8sFargateLog.LogType,
				})
			} else {
				reqBuilder.AddLogEntry(item.ID, timestamp, item.Message, ec2Event.getRegion())
			}
			continue
		}

		if reqBuilder.HasHostId() && !reqBuilder.MatchHostId(logStream) {
			output <- reqBuilder.GetLogs()
			reqBuilder = NewOtlpRequestBuilder().
				SetCloudAccount(account).
				SetLogGroup(logGroup).
				SetLogStream(logStream).
				AddLogEntry(item.ID, item.Timestamp*timestampMultiplier, item.Message, lambdaRegion)
			continue

		}

		reqBuilder.AddLogEntry(item.ID, timestamp, item.Message, lambdaRegion)
	}

	logs := reqBuilder.GetLogs()
	if logs.ResourceLogs().Len() >= 0 {
		output <- logs
	}
}

func setKubernetesInfo(reqBuilder OtlpRequestBuilder, k8sFargateLog *cloudInsightsAppLog) OtlpRequestBuilder {
	return reqBuilder.
		SetKubernetesPodName(k8sFargateLog.Kubernetes.PodName).
		SetKubernetesNamespaceName(k8sFargateLog.Kubernetes.NamespaceName).
		SetKubernetesPodUID(k8sFargateLog.Kubernetes.PodID).
		SetKubernetesContainerName(k8sFargateLog.Kubernetes.ContainerName).
		SetKubernetesContainerId(k8sFargateLog.Kubernetes.DockerID).
		SetKubernetesContainerImage(k8sFargateLog.Kubernetes.ContainerImage).
		SetKubernetesClusterUid(k8sFargateLog.ClusterUID).
		SetKubernetesNodeName(k8sFargateLog.Kubernetes.Host).
		SetKubernetesPodLabels(k8sFargateLog.Kubernetes.Labels).
		SetKubernetesPodAnnotations(k8sFargateLog.Kubernetes.Annotations).
		SetKubernetesManifestVersion(k8sFargateLog.ManifestVersion, lambdaVersion).
		SetOtelAttributes(k8sFargateLog.Kubernetes.PodName, k8sFargateLog.Kubernetes.ContainerName)
}

// test path in json object for existence of property. If `value` is provided it test also for the value to equal. Examples:
// 1. "eventSource" property
// 2. "requestParameters.instancesSet"
// 3. "requestParameters.instancesSet.items"
func testJsonPath(jsonEvent map[string]interface{}, path string, values ...string) bool {
	var value string
	if len(values) > 0 {
		value = values[0]
	}

	keys := strings.Split(path, ".")

	var exists bool
	var nextMap map[string]interface{}

	nextMap = jsonEvent

	for i, key := range keys {
		val, exists := nextMap[key]
		if !exists {
			return false
		}

		if i == len(keys)-1 {
			if value != "" {
				strVal, ok := val.(string)
				if !ok || strVal != value {
					return false
				}
			}
			return true
		}

		if nextValue, ok := val.(map[string]interface{}); ok {
			nextMap = nextValue
		} else {
			return false
		}
	}

	return exists
}

func parseMessage(message string) (ok bool, result iEc2Event) {
	ok = false
	result = nil

	var jsonEvent map[string]interface{}
	err := json.Unmarshal([]byte(message), &jsonEvent)
	if err != nil {
		ok = false
		return
	}

	if testJsonPath(jsonEvent, "eventSource", "ec2.amazonaws.com") && (testJsonPath(jsonEvent, "requestParameters.instancesSet") || testJsonPath(jsonEvent, "responseElements.instancesSet")) {
		ec2Event := ec2CloudTrailEvent{}
		err := json.Unmarshal([]byte(message), &ec2Event)
		if err == nil {
			ok = true
			result = &ec2Event
			return
		}
	}

	if testJsonPath(jsonEvent, "eventVersion") {
		genericCloudTrailEvent := cloudTrailEvent{}
		err := json.Unmarshal([]byte(message), &genericCloudTrailEvent)
		if err == nil {
			ok = true
			result = &genericCloudTrailEvent
			return
		}
	}

	if testJsonPath(jsonEvent, "ec2_instance_id") {
		ciLog := cloudInsightsLog{}
		err := json.Unmarshal([]byte(message), &ciLog)
		if err == nil {
			ok = true
			result = &ciLog
			return
		}
	}

	if testJsonPath(jsonEvent, "kubernetes.host") && testJsonPath(jsonEvent, "kubernetes.namespace_name") {
		ciAppLog := cloudInsightsAppLog{}
		err := json.Unmarshal([]byte(message), &ciAppLog)
		if err == nil {
			ciAppLog.parse()
			ok = true
			result = &ciAppLog
			return
		}
	}

	if testJsonPath(jsonEvent, "InstanceId") && testJsonPath(jsonEvent, "AutoScalingGroupName") {
		ciPerfLog := cloudInsightsPerformance{}
		err := json.Unmarshal([]byte(message), &ciPerfLog)
		if err == nil {
			ciPerfLog.parse()
			ok = true
			result = &ciPerfLog
			return
		}
	}
	return
}

func (evt *ec2CloudTrailEvent) getInstanceId() (result string, err error) {
	result, err = extractEC2InstanceId(evt)
	return
}

func (evt *ec2CloudTrailEvent) getRegion() (result string) {
	result = evt.Region
	return
}

func (evt *ec2CloudTrailEvent) getEventType() (result string) {
	result = ec2Event
	return
}

func (evt *cloudInsightsLog) getInstanceId() (result string, err error) {
	result = evt.Ec2InstanceId
	return
}

func (evt *cloudInsightsLog) getRegion() (result string) {
	result = evt.Region
	return
}

func (evt *cloudInsightsLog) getEventType() (result string) {
	result = ec2Event
	return
}

func (evt *cloudInsightsAppLog) parse() {
	matches := detectInstanceNameAndRegion.FindStringSubmatch(evt.Kubernetes.Host)
	if matches != nil {
		if fargateParamIndex < len(matches) && matches[fargateParamIndex] != "" {
			evt.parsedInstanceId = ""
		} else if instanceParamIndex < len(matches) && matches[instanceParamIndex] != "" {
			evt.parsedInstanceId = matches[instanceParamIndex]
		}

		if regionParamIndex < len(matches) && matches[regionParamIndex] != "" {
			evt.parsedRegion = matches[regionParamIndex]
		}
	}
}

func (evt *cloudInsightsAppLog) getInstanceId() (result string, err error) {
	result = evt.parsedInstanceId
	if result == "" {
		// most likely Fargate instance
		err = errors.New("Instance Id is not present")
	}
	return
}

func (evt *cloudInsightsAppLog) getEventType() (result string) {
	matches := detectInstanceNameAndRegion.FindStringSubmatch(evt.Kubernetes.Host)
	if matches != nil && fargateParamIndex < len(matches) && matches[fargateParamIndex] != "" {
		result = fargateEvent
		return
	}

	result = ec2Event
	return
}

func (evt *cloudInsightsAppLog) getRegion() (result string) {
	result = evt.parsedRegion
	return
}

func (evt *cloudInsightsPerformance) parse() {
	matches := detectInstanceNameAndRegion.FindStringSubmatch(evt.NodeName)
	if regionParamIndex < len(matches) {
		evt.parsedRegion = matches[regionParamIndex]
	}
}

func (evt *cloudInsightsPerformance) getInstanceId() (result string, err error) {
	result = evt.InstanceId
	return
}

func (evt *cloudInsightsPerformance) getRegion() (result string) {
	result = evt.parsedRegion
	return
}

func (evt *cloudInsightsPerformance) getEventType() (result string) {
	result = ec2Event
	return
}

func (evt *cloudTrailEvent) getInstanceId() (result string, err error) {
	result = ""
	err = errors.New("Event doesn't contain EC2 Instance ID")
	return
}

func (evt *cloudTrailEvent) getRegion() (result string) {
	result = evt.Region
	return
}

func (evt *cloudTrailEvent) getEventType() (result string) {
	result = ec2Event
	return
}

func main() {
	lambda.Start(handleEvent)
}
