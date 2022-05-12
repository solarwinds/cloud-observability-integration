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
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
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

type cloudTrailEvent struct {
	EventSource string `json:"eventSource"`
	EventName string `json:"eventName"`
}

type ec2InstanceParameter struct {
	InstanceId string `json:instanceId`
}
type ec2InstancesSetItems struct {
	Items [] ec2InstanceParameter `json:items`
}

type ec2InstancesSet struct {
	InstancesSet ec2InstancesSetItems `json:instancesSet`
}
type ec2CloudTrailEvent struct {
	cloudTrailEvent
	RequestParameters ec2InstancesSet `json:requestParameters`
	ResponseElements ec2InstancesSet `json:responseElements`
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
	
	defer conn.Close()

	if err != nil {
		appLogger.Error("While connecting to otlp/gRPC endpoint: ", err.Error())
		return r, err
	}

	logsClient := otlpgrpc.NewLogsClient(conn)
	logsData := pdata.NewLogs()
	resLog := logsData.ResourceLogs().AppendEmpty()
	resLog.SetSchemaUrl(semconv.SchemaURL)
	resource := resLog.Resource()
	attrs := resource.Attributes()
	attrs.InsertString(semconv.AttributeCloudProvider, semconv.AttributeCloudProviderAWS)
	attrs.InsertString(semconv.AttributeCloudAccountID, datareq.Owner)
	attrs.InsertString(semconv.AttributeAWSLogGroupNames, datareq.LogGroup)
	attrs.InsertString(semconv.AttributeAWSLogStreamNames, datareq.LogStream)
	if strings.HasPrefix( datareq.LogStream, "i-") {
		// assume this log stream belongs to EC2 instance
		appLogger.Info(fmt.Sprintf("Assuming log belongs to '%s' EC2 instance", datareq.LogStream))
		attrs.InsertString(semconv.AttributeHostID, datareq.LogStream)
		attrs.InsertString(semconv.AttributeCloudPlatform, semconv.AttributeCloudPlatformAWSEC2)
	}
	instrLog := resLog.InstrumentationLibraryLogs().AppendEmpty()
	
	for _, item := range datareq.LogEvents {
		logEntry := instrLog.Logs().AppendEmpty()
		logEntry.SetName(item.ID)
		logEntry.SetTimestamp(pdata.Timestamp(item.Timestamp * timestampMultiplier))
		logEntry.Body().SetStringVal(item.Message)

		// check if message comes from EC2 CloudTrail

		if strings.Contains(item.Message, "ec2.amazonaws.com") && strings.Contains(item.Message, "instancesSet") {
			// parse message as json event object

			ec2Event := ec2CloudTrailEvent {}
			err = json.Unmarshal([]byte(item.Message), &ec2Event)
			if err == nil {
				instanceId, err := extractEC2InstanceId(&ec2Event)
				if err == nil {
					attrs.UpsertString(semconv.AttributeHostID, instanceId)
					attrs.UpsertString(semconv.AttributeCloudPlatform, semconv.AttributeCloudPlatformAWSEC2)
				}
			}
		}
	}
	
	logRequest := otlpgrpc.NewLogsRequest()
	logRequest.SetLogs(logsData)

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer " + apiToken)
	_, err = logsClient.Export(ctx, logRequest)
	
	if err != nil {
		appLogger.Error("While exporting log data: ", err.Error())
		return r, err
	}

	r = "success"
	appLogger.Info("Function execution result: ", r)
	return r, nil
}

func main() {
	lambda.Start(handleEvent)
}
