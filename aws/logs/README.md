# sendlogs

This readme provides a brief explanation of what is included in the project:

```bash
.
├── README.md                   <-- This instructions file
├── send-logs                   <-- Source code for a lambda function
│   ├── go.mod                  <-- Go dependency definitions
|   ├── go.sum                  <-- Go dependency checksums
│   ├── main.go                 <-- Lambda function code
│   ├── logger
│       ├── logger.go           <-- Simple logger utility library
└── template.yaml               <-- AWS SAM deployment template
```

## Requirements

* AWS CLI already configured with Administrator permission [getting started guide](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html)
* [AWS SAM CLI installed](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
* [Docker installed](https://www.docker.com/community-edition)
* [Golang](https://golang.org)

### Building

Golang is a statically compiled language. This means that to run it, you have to build the executable target. Ensure your `go.mod` file in your application directory has all of your dependencies and run `sam build` from the source root directory to build your application using Go modules.

### Configuration

The lambda function requires the following environment variables to be set:
* `USE_ENCRYPTION` - tells the function to decrypt environmental variables (default is `yes`, clear to turn off the encryption)
* `OTLP_ENDPOINT` - OTEL Collector logs receiver endpoint
* `API_TOKEN` - SolarWinds API token generated for the customer

### Testing

It is possible to test the lambda function locally against an OTEL Collector. Refer to this [guide](https://opentelemetry.io/docs/collector/getting-started/) and select the most appropriate option for you.
Adjust the OTEL Collector configuration to expose otlp/GRPC receiver endpoint:
```yaml
...
receivers:
  otlp:
    protocols:
      grpc:
...
```

and pipeline:
```yaml
...
  pipelines:
    logs:
      receivers: [otlp]
...
```
Make sure that the OTEL Collector listens on 4317. Map this port to your localhost if necessary.
Create a json file with environment variables, env.json
```json
{
	"SendLogsFunction" : {
		"OTLP_ENDPOINT" : "host.docker.internal:4317",
		"API_TOKEN" : "test"
	}
}
```
Note that when running the OTEL Collector with docker, you will need to map port 4317 and provide the path to your configuration file. 
```bash
docker run -p 4317:4317 -v /otel-config-folder:/otel-config  otel/opentelemetry-collector:0.32.0 --config /otel-config/config.yaml
```
## Packaging and deployment
### From Serverless Applications Repository

Navigate to `Lambda -> Functions -> Create function` in AWS Lambda Console.

Choose `Browse serverless app repository` and search for `send-logs-app` (*TODO: verify that this is the correct published app name*).

Specify `Application name` in `Application settings` if it is not already specified and click `Deploy`.

Navigate to `Lambda -> Functions` and locate your newly created function. Click `Add trigger` and specify `CloudWatch Logs`. Choose `Log group`, specify a name for the trigger and click `Add`. It is possible to specify multiple triggers as well as provide filters for log events.

Then navigate to `Configuration` tab and select `Environment variables` in the side bar. 

Provide values for `OTLP_ENDPOINT` and `API_TOKEN` variables and encrypt them with helpers for encryption in transit. For configuring the encryption, refer to the [Configuring Environment Variables](https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html#configuration-envvars-encryption) guide.

### From source code
AWS SAM will use the `CodeUri` property to know where to look for both application and dependencies:

```yaml
...
SendLogsFunction:
        Type: AWS::Serverless::Function
        Properties:
            CodeUri: send-logs/
            ...
```
#### Building and deploying the lambda function for the first time
To deploy your application for the first time, run the following code in your shell:

```bash
sam build
sam deploy --guided
```

The command will build, package, and deploy your application to AWS, with a series of prompts:

* **Stack Name**: The name of the stack to deploy to CloudFormation. This should be unique to your account and region. A good starting point would be something matching your project name.
* **AWS Region**: The AWS region you want to deploy your app to.
* **Confirm changes before deploy**: If set to yes, any change sets will be shown to you before execution for manual review. If set to no, the AWS SAM CLI will automatically deploy application changes.
* **Allow AWS SAM CLI IAM role creation**: Many AWS SAM templates, including this example, create AWS IAM roles required for the AWS Lambda function(s) included to access AWS services. By default, these are scoped down to minimum required permissions. To deploy an AWS CloudFormation stack that creates or modifies IAM roles, the `CAPABILITY_IAM` value for `capabilities` must be provided. If the permission isn't provided through this prompt, to deploy this example, you must explicitly pass `--capabilities CAPABILITY_IAM` to the `sam deploy` command.
* **Save arguments to samconfig.toml**: If set to yes, your choices will be saved to a configuration file inside the project so that in the future, you can just re-run `sam deploy` without parameters to deploy changes to your application.

#### Publishing the lambda function to the Serverless Application Repository

You can publish your application with the `Send Logs` lambda function to your private section of Serverless Application Repository. Use the following commands to do that (starting from project root):
```bash
sam build
cd .aws-sam/build
sam package --template-file template.yaml --output-template-file packaged.yaml --s3-bucket <your S3 bucket>
sam publish --template-file packaged.yaml
```
At first, the `sam build` command produces output in `.aws-sam/build` directory. The directory includes the processed `template.yaml` file and the compiled lambda function `SendLogsFunction/main`.

After building, it is necessary to change to `.aws-sam/build` before you run any subsequent commands.

The packaging command uploads the lambda function build artifacts to S3 store and produce `packaged.yaml` file with concrete links to the artifacts stored in the S3 bucket.

The publishing command deploys the `packaged.yaml` application manifest to the private section of Serverless Application Repository from which it can be deployed to the AWS account or eventually made public. 
# Appendix

### Golang installation

Please ensure that Go 1.x (where 'x' is the latest version) is installed using the instructions on the official golang website: https://golang.org/doc/install

A quick way would be to use Homebrew, chocolatey, or your linux package manager.

#### Homebrew (Mac)

Issue the following command from the terminal:

```shell
brew install golang
```

If it's already installed, run the following command to ensure it's the latest version:

```shell
brew update
brew upgrade golang
```

#### Chocolatey (Windows)

Issue the following command from the powershell:

```shell
choco install golang
```

If it's already installed, run the following command to ensure it's the latest version:

```shell
choco upgrade golang
```