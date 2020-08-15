package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// ExpenseData is the object returned with output from previous state
type ExpenseData struct {
	Result struct {
		Payload struct {
			ExpenseId string `json:"expenseId"`
		} `json:"Payload"`
	} `json:"result"`
}

// HandleRequest is function's handler
func HandleRequest(ctx context.Context, receipt ExpenseData) (string, error) {

	// Create DynamoDB client
	client := dynamodb.New(session.Must(session.NewSession()))

	tab := os.Getenv("TABLE")
	status := "Approved"

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s": {
				S: aws.String(status),
			},
		},
		TableName: aws.String(tab),
		Key: map[string]*dynamodb.AttributeValue{
			"ExpenseID": {
				S: aws.String(receipt.Result.Payload.ExpenseId),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set ExpenseStatus = :s"),
	}

	_, err := client.UpdateItem(input)
	if err != nil {
		return fmt.Sprintf("Could not update expense:  %s", receipt.Result.Payload.ExpenseId), err
	}

	return fmt.Sprintf("Successfully approved expense:  %s", receipt.Result.Payload.ExpenseId), nil

}

func main() {
	lambda.Start(HandleRequest)
}
