package qrapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
	S3Downloader *manager.Downloader
	S3Uploader   *manager.Uploader
	S3Client     *s3.Client
}

func (ss *S3Storage) DownloadToTmpFile(ctx context.Context, bucket, key string) (fs.File, error) {
	tmpFile, err := os.CreateTemp("", "email")
	if err != nil {
		return nil, fmt.Errorf("couldn't create tmp file: %s", err)
	}
	_, err = ss.S3Downloader.Download(ctx, tmpFile, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't download %s from %s: %s", bucket, key, err)
	}
	return tmpFile, nil
}

func (ss *S3Storage) RemoveTmpFile(ctx context.Context, tmpFile fs.File) error {
	if file, ok := tmpFile.(*os.File); ok {
		return os.Remove(file.Name())
	}
	return errors.New("unexpected file type")
}

func (ss *S3Storage) Upload(ctx context.Context, bucket, key, contentType string, r io.Reader) error {
	_, err := ss.S3Uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return err
	}
	return nil
}

func (ss *S3Storage) Delete(ctx context.Context, bucket, key string) error {
	_, err := ss.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	return nil
}
