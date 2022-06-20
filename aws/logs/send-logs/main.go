package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"send-logs/logger"
	"strings"

	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"go.opentelemetry.io/collector/model/pdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	awsLambdaFunctionNameVar = "AWS_LAMBDA_FUNCTION_NAME"
	awsExecutionEnvVar = "AWS_EXECUTION_ENV"
	otlpEndpointVar = "OTLP_ENDPOINT"
	apiTokenVar = "API_TOKEN"
	useEncryptionVar = "USE_ENCRYPTION"
	timestampMultiplier = 1000000 // AWS Logs timestamp is in millisends since Jan 1 , 1970, OTEL Collector timestamp is in nanoseconds 
)

var (
	runningTests = false
	functionName string = os.Getenv(awsLambdaFunctionNameVar)
	executingInAWS bool = strings.Contains(os.Getenv(awsExecutionEnvVar), "AWS_Lambda_") 
	useEncryption = executingInAWS && strings.EqualFold(os.Getenv(useEncryptionVar), "yes")
	endpoint string = os.Getenv(otlpEndpointVar) // encrypted when AWS_EXECUTION_ENV contains 'AWS_Lambda_'
	apiToken string = os.Getenv(apiTokenVar) // encrypted when AWS_EXECUTION_ENV contains 'AWS_Lambda_'
	appLogger = logger.NewLogger("send-logs")
	kmsClient *kms.KMS
)

type MessageKind int

const (
	Default MessageKind = iota
	CloudTrailEC2
)

type cloudTrailEvent struct {
	EventSource string `json:"eventSource"`
	EventName string `json:"eventName"`
}

type ec2InstanceParameter struct {
	InstanceId string `json:"instanceId"`
}
type ec2InstancesSetItems struct {
	Items [] ec2InstanceParameter `json:"items"`
}

type ec2InstancesSet struct {
	InstancesSet ec2InstancesSetItems `json:"instancesSet"`
}
type ec2CloudTrailEvent struct {
	cloudTrailEvent
	RequestParameters ec2InstancesSet `json:"requestParameters"`
	ResponseElements ec2InstancesSet `json:"responseElements"`
}

type cloudInsightsLog struct {
	Ec2InstanceId string `json:"ec2_instance_id"`  
}

type cloudInsightsAppLogKubernetes struct {
	Host string `json:"host"`
}

type cloudInsightsAppLog struct {
	Kubernetes cloudInsightsAppLogKubernetes `json:"kubernetes"`
}

type cloudInsightsPerformance struct {
	InstanceId string `json:"InstanceId"`
}


type iEc2Event interface {
	getInstanceId() (string, error)
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

func extractEC2InstanceId(ec2Event * ec2CloudTrailEvent) (instanceId string, err error) {
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
		config := &tls.Config {}
		dialOption = grpc.WithTransportCredentials(credentials.NewTLS(config))
	} 
	
	conn, err := grpc.Dial(endpoint, dialOption)
	
	if err != nil {
		appLogger.Error("While connecting to otlp/gRPC endpoint: ", err.Error())
		return r, err
	}

	defer conn.Close()

	logsClient := otlpgrpc.NewLogsClient(conn)
	logsChan := make(chan pdata.Logs)
	go transformLogEvents(datareq.Owner, datareq.LogGroup, datareq.LogStream, datareq.LogEvents, logsChan)

	errs := make([] error, 0)
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer " + apiToken)
	
	for logsData := range logsChan  {
		logRequest := otlpgrpc.NewLogsRequest()
		logRequest.SetLogs(logsData)
		_, err = logsClient.Export(ctx, logRequest)
		if err != nil {
			appLogger.Error("While exporting log data: ", err.Error())
			errs = append(errs, err)
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

func transformLogEvents(account, logGroup, logStream string, input [] events.CloudwatchLogsLogEvent, output chan pdata.Logs) {
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
					reqBuilder.AddLogEntry(item.ID, timestamp, item.Message)
					continue
				}
				if !reqBuilder.MatchHostId(instanceId) {
					output <- reqBuilder.GetLogs()
					reqBuilder = NewOtlpRequestBuilder().
						SetCloudAccount(account).
						SetLogGroup(logGroup).
						SetLogStream(logStream).
						SetHostId(instanceId).
						AddLogEntry(item.ID, timestamp, item.Message)
					continue
				}
			}
		}

		if reqBuilder.HasHostId() && !reqBuilder.MatchHostId(logStream) {
		output <- reqBuilder.GetLogs()
		reqBuilder = NewOtlpRequestBuilder().
			SetCloudAccount(account).
			SetLogGroup(logGroup).
			SetLogStream(logStream).
			AddLogEntry(item.ID, item.Timestamp * timestampMultiplier, item.Message)
			continue

		}

		reqBuilder.AddLogEntry(item.ID, timestamp, item.Message)
	}

	logs := reqBuilder.GetLogs()
	if logs.ResourceLogs().Len() >= 0 {
		output <- logs
	}
}

func parseMessage(message string) (ok bool, result iEc2Event) {
	ok = false
	result = nil
	if strings.Contains(message, "ec2.amazonaws.com") && strings.Contains(message, "instancesSet") {
		// parse message as json event object

		ec2Event := ec2CloudTrailEvent {}
		err := json.Unmarshal([]byte(message), &ec2Event)
		if err == nil {
			ok = true
			result = &ec2Event
			return
		}
	}
	if strings.Contains(message, "ec2_instance_id") {
		ciLog := cloudInsightsLog{}
		err := json.Unmarshal([]byte(message), &ciLog)
		if err == nil {
			ok = true
			result = &ciLog
			return
		}
	}

	if strings.Contains(message, "host") &&  strings.Contains(message, "namespace_name") {
		ciAppLog := cloudInsightsAppLog{}
		err := json.Unmarshal([]byte(message), &ciAppLog)
		if err == nil {
			ok = true
			result = &ciAppLog
			return
		}
	}

	if strings.Contains(message, "InstanceId") &&  strings.Contains(message, "AutoScalingGroupName") {
		ciPerfLog := cloudInsightsPerformance{}
		err := json.Unmarshal([]byte(message), &ciPerfLog)
		if err == nil {
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

func (evt *cloudInsightsLog) getInstanceId() (result string, err error) {
	result = evt.Ec2InstanceId
	return
}

func (evt *cloudInsightsAppLog) getInstanceId() (result string, err error) {
	
	if strings.HasPrefix( evt.Kubernetes.Host, "i-") {
		pos := strings.Index(evt.Kubernetes.Host, ".")
		if pos < 0 {
			pos = len(evt.Kubernetes.Host)
		}
		result = evt.Kubernetes.Host[0: pos]
	} else {
		err = errors.New(fmt.Sprint("Failed to parse EC2 instance ID from ", evt.Kubernetes.Host))
	}
	return
}


func (evt *cloudInsightsPerformance) getInstanceId() (result string, err error) {
	result = evt.InstanceId
	return
}
func main() {
	lambda.Start(handleEvent)
}
