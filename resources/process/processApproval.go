package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
)

type EventData struct {
	Query struct {
		Action       string `json:"action"`
		TaskToken    string `json:"taskToken"`
		StateMachine string `json:"sm"`
		Execution    string `json:"ex"`
		ExpenseId    string `json:"expenseid"`
	} `json:"query"`
}

type ExpenseData struct {
	ExpenseStatus string `json:"expenseStatus"`
	Payload       struct {
		ExpenseId string `json:"expenseId"`
		Alert     string `json:"alert"`
	} `json:"Payload"`
}

func HandleRequest(ctx context.Context, data EventData) (ExpenseData, error) {

	client := sfn.New(session.Must(session.NewSession()))
	res := ExpenseData{}

	if data.Query.Action == "approve" {
		res.ExpenseStatus = "Expense is approved"
	} else if data.Query.Action == "reject" {
		res.ExpenseStatus = "Expense is rejected"
		res.Payload.Alert = "Expense report was rejected. Please reach out to the approver for further clarifications."
	} else {
		return res, errors.New("Unrecognized action. Expected operations: approve, reject.")
	}

	res.Payload.ExpenseId = data.Query.ExpenseId

	output, _ := json.Marshal(res)

	params := &sfn.SendTaskSuccessInput{
		Output:    aws.String(string(output)),
		TaskToken: aws.String(data.Query.TaskToken),
	}
	_, err := client.SendTaskSuccess(params)

	if err != nil {
		return res, err
	}

	return res, nil
}

func main() {
	lambda.Start(HandleRequest)
}
