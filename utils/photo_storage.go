package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func photoS3() (*s3.Client, error) {
	region := getS3Region()
	endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	options := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if endpoint != "" {
		options = append(options, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), "")))
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), options...)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	}), nil
}

func photoBucket(export bool) (string, error) {
	key, fallback := "PHOTO_S3_BUCKET", "event-photos"
	if export {
		key, fallback = "PHOTO_EXPORT_S3_BUCKET", "event-photo-exports"
	}
	bucket := strings.TrimSpace(os.Getenv(key))
	if bucket == "" {
		bucket = fallback
	}
	if bucket == strings.TrimSpace(os.Getenv("S3_BUCKET")) {
		return "", errors.New("photo storage must not use the public bucket")
	}
	return bucket, nil
}

func UploadPrivatePhoto(ctx context.Context, key string, body io.Reader, contentType string, export bool) error {
	bucket, err := photoBucket(export)
	if err != nil {
		return err
	}
	client, err := photoS3()
	if err != nil {
		return err
	}
	_, err = transfermanager.New(client).UploadObject(ctx, &transfermanager.UploadObjectInput{Bucket: aws.String(bucket), Key: aws.String(key), Body: body, ContentType: aws.String(contentType)})
	return err
}

func DeletePrivatePhoto(ctx context.Context, key string, export bool) error {
	bucket, err := photoBucket(export)
	if err != nil {
		return err
	}
	client, err := photoS3()
	if err != nil {
		return err
	}
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	return err
}

func ReadPrivatePhoto(ctx context.Context, key string, export bool) (io.ReadCloser, error) {
	bucket, err := photoBucket(export)
	if err != nil {
		return nil, err
	}
	client, err := photoS3()
	if err != nil {
		return nil, err
	}
	result, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func SignPrivatePhoto(ctx context.Context, key string, export bool, duration time.Duration) (string, error) {
	bucket, err := photoBucket(export)
	if err != nil {
		return "", err
	}
	client, err := photoS3()
	if err != nil {
		return "", err
	}
	request, err := s3.NewPresignClient(client).PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}, func(o *s3.PresignOptions) { o.Expires = duration })
	if err != nil {
		return "", fmt.Errorf("sign private object: %w", err)
	}
	return request.URL, nil
}
