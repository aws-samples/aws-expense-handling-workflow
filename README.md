# Workflow orchestration with AWS CDK - An Expense Handling Scenario
Workflow orchestration with CDK and Step Functions is a developer-friendly and efficient way to build and automate your business processes on AWS using your favourite programming languages. 
In this project we build the process orchestration layer, construct the necessary AWS services and develop the code containing the business logic all in the same codebase.

We use AWS Java CDK for the workflow and service constructs and Go for the business logic part.

Services used are AWS Step Functions, AWS Lambda, Amazon S3, Amazon API Gateway, Amazon Textract, Amazon DynamoDB, AWS CloudTrail, Amazon CloudWatch and Amazon SNS.


## Workflow
The expense handling workflow automates the process of submitting,registering and approving expenses. This automatic process is extended with a manual approval workflow in case of exceptions.

The workflow is captured in the following state machine:
![Expense Handling Workflow](https://raw.githubusercontent.com/evisb/aws-expense-handling-workflow/master/images/workflow.png)

The activities performed in the process are the following:

- Receipts are uploaded to S3, after which they get processed and validated.
- The submitter is notified in case the content of the receipt isn't properly detected and as a consequence cannot be processed.
- If the processing is successful and the validation criteria have been met, the expense is registered in the back-end system.
- Expenses get auto-approved if a certain monetary threshold hasn't been reached.
- All other expenses above the threshold are forwarded for approval to a manual workflow.
- Once the expense is either approved or rejected via a callback mechanism, the expense gets approved in the system or alternatively the submitter is notified about the rejection.


## Prerequisites
- AWS CDK installed and configured. Bootstrap your AWS account if you already haven't done so. Read section "Bootstrapping your AWS environment" in [AWS CDK Toolkit](https://docs.aws.amazon.com/cdk/latest/guide/cli.html)
- AWS credentials profile and default region set locally
- Java Runtime and Development Kit 11+
- Apache Maven: [Installing Apache Maven](https://maven.apache.org/install.html)
- Go 1.12+: [Installing Go](https://golang.org/doc/install). GOPATH configuration is required: [Setting GOPATH environment variable](https://github.com/golang/go/wiki/SettingGOPATH)

## Getting Started
To compile the business services written in Go run the following commands from the `resources` directory:
- `make dep` to get the Go dependencies
- `make build` to compile the code
- `make clean` to removed the compiled binaries 

Do not forget to replace the default email aliases `email-alias` and domains `@email-domain.extension` in the main CDK stack file `ExpenseHandlingWorkflowStack.java` for the expense submitter as well as the expense approver part.


To build the AWS services and the workflow written with AWS Java SDK, run the following command from the project root directory:
- `mvn versions:use-latest-releases`  to use the latest version of dependencies in Maven
- `mvn clean compile`                 to compile the code
- `cdk deploy`                        to deploy the stack to your default AWS account/region


## Using the service
To test the deployed workflow, use the sample receipts situated in the `samples` directory of the project and trigger the process by uploading them to the `expensehandlingworkflows-expensesrepositoryxxxxx-xxxxxxxxxxx` S3 bucket.
To check status of expenses, check the record in the Expenses DynamoDB table `ExpenseHandlingWorkflowStack-Expensesxxxxxx-xxxxxxxx`. Emails are only sent out in case of errors or rejections.

## Cleanup
To clean up the project run:

`cdk destroy`

Data generated and stored within this project will not be deleted automatically. This is to prevent unwanted removal in case the data are still needed.

To remove these data, delete the S3 buckets (`expensehandlingworkflows-expensesrepositoryxxxxx-xxxxxxxxxxx` and `expensehandlingworkflows-workflowcloudtrailbucket-xxxxxxxxxxxxx`), remove the `ExpenseHandlingWorkflowStack-Expensesxxxxxxxx-xxxxxxxxxxxxx` DynamoDB table and delete all the `/aws/lambda/ExpenseHandlingWorkflowSt-xxxxxxxxxxxx-xxxxxxxx` CloudWatch log groups.


