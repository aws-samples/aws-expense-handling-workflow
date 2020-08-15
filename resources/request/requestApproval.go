package main

import (
	"context"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

type PayloadData struct {
	Input struct {
		Result struct {
			Payload struct {
				ExpenseId string `json:"expenseId"`
			} `json:"Payload"`
		} `json:"result"`
	} `json:"input"`
	ExecutionContext struct {
		Execution struct {
			Name string `json:"Name"`
		} `json:"Execution"`
		StateMachine struct {
			Name string `json:"Name"`
		} `json:"StateMachine"`
		Task struct {
			Token string `json:"Token"`
		} `json:"Task"`
	} `json:"ExecutionContext"`
	APIGatewayEndpoint string `json:"APIGatewayEndpoint"`
}

type FunctionResponse struct {
	Result struct {
		Payload struct {
			ExpenseId string `json:"expenseId"`
		} `json:"Payload"`
	} `json:"result"`
}

func HandleRequest(ctx context.Context, data PayloadData) (FunctionResponse, error) {
	approveEndpoint := data.APIGatewayEndpoint + "execution?action=approve&ex=" + data.ExecutionContext.Execution.Name + "&expenseid=" + data.Input.Result.Payload.ExpenseId + "&sm=" + data.ExecutionContext.StateMachine.Name + "&taskToken=" + url.QueryEscape(data.ExecutionContext.Task.Token)
	rejectEndpoint := data.APIGatewayEndpoint + "execution?action=reject&ex=" + data.ExecutionContext.Execution.Name + "&expenseid=" + data.Input.Result.Payload.ExpenseId + "&sm=" + data.ExecutionContext.StateMachine.Name + "&taskToken=" + url.QueryEscape(data.ExecutionContext.Task.Token)
	emailSnsTopic := os.Getenv("TOPIC")

	res := FunctionResponse{}

	notificationMessage := "Hello! \n\n"
	notificationMessage += "This is an email requiring your approval for a submitted expense. \n\n"
	notificationMessage += "Please verify the related expense information and approve or reject the expense by respectively clicking on the \"Approve\" or \"Reject\" link. \n\n"
	notificationMessage += "Process execution ID -> " + data.ExecutionContext.Execution.Name + "\n\n"
	notificationMessage += "Expense ID -> " + data.Input.Result.Payload.ExpenseId + "\n\n"
	notificationMessage += "Approve " + approveEndpoint + "\n\n"
	notificationMessage += "Reject  " + rejectEndpoint + "\n\n"

	client := sns.New(session.Must(session.NewSession()))

	message := sns.PublishInput{
		Message:  aws.String(notificationMessage),
		Subject:  aws.String("Required approval for a submitted expense report"),
		TopicArn: aws.String(emailSnsTopic),
	}

	res.Result.Payload.ExpenseId = data.Input.Result.Payload.ExpenseId

	_, err := client.Publish(&message)
	if err != nil {
		return res, err
	}

	return res, nil

}

func main() {
	lambda.Start(HandleRequest)
}
