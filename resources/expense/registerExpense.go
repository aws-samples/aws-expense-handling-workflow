package main

import (
	"context"
	"os"
	"time"

	"github.com/gofrs/uuid"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// ReceiptData is the object returned with output from previous state
type ReceiptData struct {
	Detail struct {
		RequestParameters struct {
			BucketName string `json:"bucketName"`
			Receipt    string `json:"key"`
		} `json:"requestParameters"`
		UserIdentity struct {
			UserName string `json:"userName"`
		} `json:"userIdentity"`
	} `json:"detail"`
	Result struct {
		Payload struct {
			SubTotal float64 `json:"subTotal"`
			Tax      float64 `json:"tax"`
			Total    float64 `json:"total"`
		} `json:"Payload"`
	} `json:"result"`
}

//Expense contains expense data to be registered in the database
type Expense struct {
	ExpenseID       string
	Timestamp       string
	UserID          string
	Subtotal        float64
	Tax             float64
	Total           float64
	ReceiptLocation string
	ExpenseStatus   string
}

//FunctionResponse contains function output
type FunctionResponse struct {
	RegisterExpenseTaskStatus   string `json:"registerExpenseTaskStatus"`
	RegisterExpenseErrorMessage string `json:"registerExpenseErrorMessage"`
	ExpenseId                   string `json:"expenseId"`
}

// HandleRequest is function's handler
func HandleRequest(ctx context.Context, receipt ReceiptData) (FunctionResponse, error) {
	rec := Expense{
		ExpenseID:       uuid.Must(uuid.NewV4()).String(),
		Timestamp:       time.Now().String(),
		UserID:          receipt.Detail.UserIdentity.UserName,
		Subtotal:        receipt.Result.Payload.SubTotal,
		Tax:             receipt.Result.Payload.Tax,
		Total:           receipt.Result.Payload.Total,
		ReceiptLocation: "s3://" + receipt.Detail.RequestParameters.BucketName + "/" + receipt.Detail.RequestParameters.Receipt,
		ExpenseStatus:   "Unapproved",
	}

	res := FunctionResponse{}

	// Create DynamoDB client
	client := dynamodb.New(session.Must(session.NewSession()))

	// Marshall the data
	row, err := dynamodbattribute.MarshalMap(rec)
	if err != nil {
		res.RegisterExpenseErrorMessage = "Row could not be parsed: " + err.Error()
		res.RegisterExpenseTaskStatus = ""
		return res, err
	}

	tab := os.Getenv("TABLE")

	entry := &dynamodb.PutItemInput{
		Item:      row,
		TableName: &tab,
	}

	_, err = client.PutItem(entry)
	if err != nil {
		res.RegisterExpenseErrorMessage = "Failed to insert record in the database: " + err.Error()
		res.RegisterExpenseTaskStatus = ""
		return res, err
	}

	res.ExpenseId = rec.ExpenseID

	if rec.Total > 50 {
		res.RegisterExpenseTaskStatus = ">50"
	} else {
		res.RegisterExpenseTaskStatus = "<=50"
	}

	return res, nil
}

func main() {
	lambda.Start(HandleRequest)
}
