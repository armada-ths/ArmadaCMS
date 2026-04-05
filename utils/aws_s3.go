package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var ErrUnsupportedImageFormat = errors.New("unsupported image format")

var allowedImageContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

func getAWSRegion() string {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return "eu-north-1"
	}

	return region
}

func detectAndValidateImageContentType(file multipart.File) (string, error) {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to read uploaded file: %w", err)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to rewind uploaded file: %w", err)
	}

	contentType := http.DetectContentType(buf[:n])
	if _, ok := allowedImageContentTypes[contentType]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedImageFormat, contentType)
	}

	return contentType, nil
}

func UploadToS3(file multipart.File, header *multipart.FileHeader) (string, error) {
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	contentType, err := detectAndValidateImageContentType(file)
	if err != nil {
		return "", err
	}

	region := getAWSRegion()
	endpoint := os.Getenv("S3_ENDPOINT")
	bucket := os.Getenv("S3_BUCKET")

	var cfg aws.Config
	if endpoint != "" {
		// Local S3-compatible store (e.g. MinIO): use static credentials and custom endpoint.
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"),
				"",
			)),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(region),
		)
	}
	if err != nil {
		return "", fmt.Errorf("unable to load SDK config, %v", err)
	}

	// Create S3 client; path-style addressing is required for MinIO.
	var client *s3.Client
	if endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	// Check if the file already exists in the bucket.
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	}
	_, err = client.HeadObject(context.TODO(), headInput)
	log.Println(filename)
	if err == nil {
		return "", fmt.Errorf("file with the name %s already exists", filename)
	}

	// Upload the file.
	uploadInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        file,
		ContentType: aws.String(contentType),
	}
	_, err = client.PutObject(context.TODO(), uploadInput)
	if err != nil {
		return "", fmt.Errorf("unable to upload file to S3, %v", err)
	}

	// Build the public URL for the uploaded object.
	// S3_PUBLIC_URL separates the internal endpoint (used by the server) from the
	// browser-accessible URL (e.g. http://localhost:9000 vs http://minio:9000).
	if publicURL := os.Getenv("S3_PUBLIC_URL"); publicURL != "" {
		return fmt.Sprintf("%s/%s/%s", publicURL, bucket, filename), nil
	}
	if endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", endpoint, bucket, filename), nil
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, filename), nil
}
