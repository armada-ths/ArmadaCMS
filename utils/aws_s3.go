package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var ErrUnsupportedImageFormat = errors.New("unsupported image format")
var ErrFileTooLarge = errors.New("uploaded file exceeds maximum allowed size")

const maxImageUploadBytes = 15 * 1024 * 1024 // 15 MB

var allowedImageContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

type s3UploadTarget struct {
	bucket          string
	region          string
	endpoint        string
	publicBaseURL   string
	accessKeyID     string
	secretAccessKey string
}

func getS3Region() string {
	if r := strings.TrimSpace(os.Getenv("S3_REGION")); r != "" {
		return r
	}
	if r := strings.TrimSpace(os.Getenv("AWS_REGION")); r != "" {
		return r
	}
	return "eu-north-1"
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

func UploadImage(file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > maxImageUploadBytes {
		return "", fmt.Errorf("%w: %d bytes (max %d)", ErrFileTooLarge, header.Size, maxImageUploadBytes)
	}

	filename := buildUploadedObjectName(header.Filename)
	contentType, err := detectAndValidateImageContentType(file)
	if err != nil {
		return "", err
	}

	return uploadWithS3CompatibleBackend(file, filename, contentType, resolveS3UploadTarget())
}

func buildUploadedObjectName(originalFilename string) string {
	// Keep only printable ASCII characters; replace spaces with underscores and
	// drop anything else (non-ASCII, control chars) so the resulting S3 key is
	// always a valid, unambiguous URL path segment.
	var b strings.Builder
	for _, r := range strings.TrimSpace(originalFilename) {
		switch {
		case r == ' ':
			b.WriteByte('_')
		case r < 128 && unicode.IsPrint(r) && r != '%' && r != '#' && r != '?':
			b.WriteRune(r)
		}
	}
	cleanedName := b.String()
	if cleanedName == "" {
		cleanedName = "upload"
	}

	return fmt.Sprintf("%d_%s", time.Now().UnixNano(), cleanedName)
}

func resolveS3UploadTarget() s3UploadTarget {
	return s3UploadTarget{
		bucket:          os.Getenv("S3_BUCKET"),
		region:          getS3Region(),
		endpoint:        os.Getenv("S3_ENDPOINT"),
		publicBaseURL:   strings.TrimRight(firstNonEmpty(os.Getenv("S3_PUBLIC_URL"), os.Getenv("S3_ENDPOINT")), "/"),
		accessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		secretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}
}

func uploadWithS3CompatibleBackend(file multipart.File, filename string, contentType string, target s3UploadTarget) (string, error) {
	if target.bucket == "" {
		return "", errors.New("storage bucket is not configured")
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to rewind uploaded file: %w", err)
	}

	var cfg aws.Config
	var err error
	if target.endpoint != "" {
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(target.region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				target.accessKeyID,
				target.secretAccessKey,
				"",
			)),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(target.region),
		)
	}
	if err != nil {
		return "", fmt.Errorf("unable to load SDK config, %v", err)
	}

	var client *s3.Client
	if target.endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(target.endpoint)
			o.UsePathStyle = true
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(target.bucket),
		Key:    aws.String(filename),
	}
	_, err = client.HeadObject(context.TODO(), headInput)
	if err == nil {
		return "", fmt.Errorf("file with the name %s already exists", filename)
	}

	tm := transfermanager.New(client)
	_, err = tm.UploadObject(context.TODO(), &transfermanager.UploadObjectInput{
		Bucket:      aws.String(target.bucket),
		Key:         aws.String(filename),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("unable to upload file to S3, %v", err)
	}

	if target.publicBaseURL != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimRight(target.publicBaseURL, "/"), target.bucket, filename), nil
	}
	if target.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimRight(target.endpoint, "/"), target.bucket, filename), nil
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", target.bucket, target.region, filename), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}
