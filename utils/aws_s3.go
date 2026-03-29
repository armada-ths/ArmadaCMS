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
	// Load AWS configuration
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	contentType, err := detectAndValidateImageContentType(file)
	if err != nil {
		return "", err
	}

	region := getAWSRegion()

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
	)

	if err != nil {
		return "", fmt.Errorf("unable to load SDK config, %v", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// Check if the file already exists in the S3 bucket
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(filename),
	}
	_, err = client.HeadObject(context.TODO(), headInput)
	log.Println(filename)
	if err == nil {
		return "", fmt.Errorf("file with the name %s already exists", filename)
	}

	// Create the S3 upload request
	uploadInput := &s3.PutObjectInput{
		Bucket:      aws.String(os.Getenv("S3_BUCKET")),
		Key:         aws.String(filename), // The file name (or path) in the S3 bucket
		Body:        file,
		ContentType: aws.String(contentType),
	}

	// Upload to S3
	_, err = client.PutObject(context.TODO(), uploadInput)
	if err != nil {
		return "", fmt.Errorf("unable to upload file to S3, %v", err)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", os.Getenv("S3_BUCKET"), region, filename), nil
}
