package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ses"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jriquelme/home-it-services/qrapp"
)

func main() {
	// get env variables
	filesBucket := os.Getenv("FILES_BUCKET")
	if filesBucket == "" {
		log.Fatalf("missing FILES_BUCKET")
	}
	// load aws config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	// run lambda
	s3Cli := s3.NewFromConfig(cfg)
	storage := &qrapp.S3Storage{
		S3Downloader: manager.NewDownloader(s3Cli),
		S3Uploader:   manager.NewUploader(s3Cli),
		S3Client:     s3.NewFromConfig(cfg),
	}
	mailer := &qrapp.SESMailer{
		SESClient: ses.NewFromConfig(cfg),
	}
	app := &qrapp.QRApp{
		Storage:        storage,
		Mailer:         mailer,
		FilesBucket:    filesBucket,
		FilesBucketURL: "http://" + filesBucket,
	}
	lambda.Start(func(ctx context.Context, event events.SNSEvent) error {
		for _, record := range event.Records {
			var msg qrapp.Message
			err := json.Unmarshal([]byte(record.SNS.Message), &msg)
			if err != nil {
				log.Printf("couldn't unmarshal, discarding:\n%s\nerror: %s", record.SNS.Message, err)
				continue
			}
			log.Printf("processing email from:%s subject:%s", msg.Mail.CommonHeaders.From, msg.Mail.CommonHeaders.Subject)
			err = app.ProcessEmail(ctx, &msg)
			if err != nil {
				log.Printf("error processing email: %s", err)
			}
		}
		return nil
	})
}
