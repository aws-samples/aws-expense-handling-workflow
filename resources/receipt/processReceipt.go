package main

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/textract"
)

// RaisedEvent contains the input payload from the S3 Event rule
type RaisedEvent struct {
	Detail struct {
		RequestParameters struct {
			BucketName string `json:"bucketName"`
			Receipt    string `json:"key"`
		} `json:"requestParameters"`
	} `json:"detail"`
}

//FunctionResponse contains function output
type FunctionResponse struct {
	ProcessReceiptTaskStatus string  `json:"processReceiptTaskStatus"`
	Alert                    string  `json:"alert"`
	Subtotal                 float64 `json:"subTotal"`
	Tax                      float64 `json:"tax"`
	Total                    float64 `json:"total"`
}

// HandleRequest is function's handler
func HandleRequest(ctx context.Context, eventInput RaisedEvent) (FunctionResponse, error) {

	res := FunctionResponse{}
	var label string

	client := textract.New(session.Must(session.NewSession()))

	var v1 []*string
	tab := "TABLES"
	form := "FORMS"
	v1 = append(v1, &tab, &form)

	docInput := &textract.AnalyzeDocumentInput{}
	docInput.SetDocument(
		&textract.Document{}).SetFeatureTypes(v1)
	docInput.Document.SetS3Object(
		&textract.S3Object{
			Bucket: &eventInput.Detail.RequestParameters.BucketName,
			Name:   &eventInput.Detail.RequestParameters.Receipt})

	output, err := client.AnalyzeDocument(docInput)
	if err != nil {
		res.Alert = "Document could not be analyzed: " + err.Error()
		res.ProcessReceiptTaskStatus = "Failure"
		return res, nil
	}

	for _, v := range output.Blocks {
		if *v.BlockType == "CELL" {
			for _, r := range v.Relationships {
				for _, wordid := range r.Ids {
					for _, w := range output.Blocks {
						if *w.BlockType == "WORD" && *w.Id == *wordid {

							ret, _ := regexp.MatchString(`[\$£€]?(([1-9]\d{0,2}(,\d{3})*)|0)?\.\d{1,2}[\$£€]?$`, *w.Text)

							if strings.Contains(strings.ToLower(*w.Text), "subtotal") {
								label = "Subtotal"
							} else if strings.Contains(strings.ToLower(*w.Text), "tax") {
								label = "Tax"
							} else if strings.Contains(strings.ToLower(*w.Text), "total") {
								label = "Total"
							} else if !(label != "" && (strings.ContainsAny(*w.Text, "$£€") || ret == true)) {
								label = ""
							}

							if ret == true {
								if label != "" {
									amount, _ := strconv.ParseFloat(strings.Trim(*w.Text, "$£€"), 64)
									if label == "Subtotal" {
										res.Subtotal = amount
									} else if label == "Tax" {
										res.Tax = amount
									} else if label == "Total" {
										res.Total = amount
									}
								}
							}

						}

					}
				}
			}
		}
	}

	if res.Total > 0 && res.Total == res.Subtotal+res.Tax {
		res.ProcessReceiptTaskStatus = "Success"
	} else {
		res.ProcessReceiptTaskStatus = "Failure"
		res.Alert = "Receipt: " + eventInput.Detail.RequestParameters.Receipt + " is not valid. Please provide a legible receipt or use the system to manually register the expense"
	}
	return res, nil
}

func main() {
	lambda.Start(HandleRequest)
}
