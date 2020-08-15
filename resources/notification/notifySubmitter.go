package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

// AlertData is the object returning the notifcation message describing the expense issue to the submitter
type AlertData struct {
	Result struct {
		Payload struct {
			Alert string `json:"alert"`
		} `json:"Payload"`
	} `json:"result"`
}

func HandleRequest(ctx context.Context, notification AlertData) (string, error) {

	emailSnsTopic := os.Getenv("TOPIC")

	notificationMessage := "Hello \n\n"
	notificationMessage += notification.Result.Payload.Alert + "\n\n"
	notificationMessage += "Thank you. \n\n"

	client := sns.New(session.Must(session.NewSession()))

	message := sns.PublishInput{
		Message:  aws.String(notificationMessage),
		Subject:  aws.String("Submitted expense report was not processed"),
		TopicArn: aws.String(emailSnsTopic),
	}

	_, err := client.Publish(&message)
	if err != nil {
		return fmt.Sprintf("Failed to send out notification to the expense submitter."), err
	}

	return fmt.Sprintf("Notification to the expense submitter was sent out successfully."), nil

}

func main() {
	lambda.Start(HandleRequest)
}
