//go:build integration

package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestUploadImageToLocalSupabaseStorage(t *testing.T) {
	if os.Getenv("RUN_S3_INTEGRATION") != "1" {
		t.Skip("set RUN_S3_INTEGRATION=1 to test local Supabase Storage")
	}
	setIntegrationDefault(t, "S3_BUCKET", "armadacms-files")
	setIntegrationDefault(t, "S3_ENDPOINT", "http://127.0.0.1:54321/storage/v1/s3")
	setIntegrationDefault(t, "S3_PUBLIC_URL", "http://127.0.0.1:54321/storage/v1/object/public")
	setIntegrationDefault(t, "S3_REGION", "local")
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Fatal("AWS_ACCESS_KEY_ID must contain S3_PROTOCOL_ACCESS_KEY_ID from supabase status -o env")
	}
	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		t.Fatal("AWS_SECRET_ACCESS_KEY must contain S3_PROTOCOL_ACCESS_KEY_SECRET from supabase status -o env")
	}

	// A valid 1x1 transparent PNG keeps the fixture small while exercising the
	// same MIME validation and upload path as application images.
	pngBytes, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode PNG fixture: %v", err)
	}

	file, err := os.CreateTemp(t.TempDir(), "armadacms-storage-*.png")
	if err != nil {
		t.Fatalf("create temporary image: %v", err)
	}
	defer file.Close()

	if _, err := file.Write(pngBytes); err != nil {
		t.Fatalf("write temporary image: %v", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("rewind temporary image: %v", err)
	}

	publicURL, err := UploadImage(file, &multipart.FileHeader{
		Filename: "storage-integration.png",
		Size:     int64(len(pngBytes)),
	})
	if err != nil {
		t.Fatalf("upload image: %v", err)
	}

	objectURL, err := url.Parse(publicURL)
	if err != nil {
		t.Fatalf("parse uploaded object URL: %v", err)
	}
	objectKey := path.Base(objectURL.Path)
	t.Cleanup(func() {
		deleteIntegrationObject(t, objectKey)
	})

	response, err := http.Get(publicURL)
	if err != nil {
		t.Fatalf("retrieve uploaded image: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("retrieve uploaded image: got HTTP %d", response.StatusCode)
	}
	retrievedBytes, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read uploaded image: %v", err)
	}
	if !bytes.Equal(retrievedBytes, pngBytes) {
		t.Fatal("retrieved image does not match uploaded image")
	}
}

func setIntegrationDefault(t *testing.T, name string, value string) {
	t.Helper()
	if os.Getenv(name) == "" {
		t.Setenv(name, value)
	}
}

func deleteIntegrationObject(t *testing.T, objectKey string) {
	t.Helper()
	target := resolveS3UploadTarget()

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(target.region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			target.accessKeyID,
			target.secretAccessKey,
			"",
		)),
	)
	if err != nil {
		t.Errorf("load S3 config for cleanup: %v", err)
		return
	}

	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(target.endpoint)
		options.UsePathStyle = true
	})
	if _, err := client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(target.bucket),
		Key:    aws.String(objectKey),
	}); err != nil {
		t.Errorf("delete integration-test object: %v", err)
	}
}
