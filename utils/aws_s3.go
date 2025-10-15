package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
)

func UploadToS3(file io.Reader, header *multipart.FileHeader) (string, error) {
	// Load AWS configuration
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion("eu-north-1"),
	)

	if err != nil {
		return "", fmt.Errorf("unable to load SDK config, %v", err)
	}
	// cfg, err := config.LoadDefaultConfig(context.TODO(),
	// 	config.WithSharedConfigProfile("armada-prod"),
	// )
	// if err != nil {
	// 	return "", fmt.Errorf("unable to load SDK config, %v", err)
	// }

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
		ContentType: aws.String("image/jpg"), // Or adjust based on the file type
	}

	// Upload to S3
	_, err = client.PutObject(context.TODO(), uploadInput)
	if err != nil {
		return "", fmt.Errorf("unable to upload file to S3, %v", err)
	}
	return fmt.Sprintf("https://%s.s3.eu-north-1.amazonaws.com/%s", os.Getenv("S3_BUCKET"), filename), nil
}
