package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/ses"
)

type QRApp struct {
	SESClient *ses.Client
}

func (q *QRApp) Handler(ctx context.Context, event events.SNSEvent) error {
	log.Printf("event: %+v", event)
	for _, record := range event.Records {
		log.Printf("MSG->%s", record.SNS.Message)
	}
	return nil
}
