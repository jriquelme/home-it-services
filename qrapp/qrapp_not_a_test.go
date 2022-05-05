//go:build dev

package qrapp

import (
	"context"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotATest is... not a test :D
// This is just an easy and comfortable way to run QRApp with existing resources from AWS (SES config, S3 buckets, etc.)
// That's why is tagged with dev and doesn't run from the Makefile.
// I know, I should create some integration tests and bla bla
func TestNotATest(t *testing.T) {
	// load aws config
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("larix"),
		config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	// configure app
	s3Client := s3.NewFromConfig(cfg)
	app := &QRApp{
		Storage: &S3Storage{
			S3Downloader: manager.NewDownloader(s3Client),
			S3Uploader:   manager.NewUploader(s3Client),
			S3Client:     s3Client,
		},
		Mailer: &SESMailer{
			SESClient: ses.NewFromConfig(cfg),
		},
		FilesBucket:    "qr.larix.cl",
		FilesBucketURL: "http://qr.larix.cl",
	}
	// run with email
	msg, err := testingMsg("snsemail-multiple-attachments-no-bkg.json")
	require.Nil(t, err)
	err = app.ProcessEmail(context.Background(), msg)
	assert.Nil(t, err)
}
