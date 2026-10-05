package utils

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type uploadTestFile struct{ *bytes.Reader }

func (uploadTestFile) Close() error { return nil }

func TestUploadStorageErrorRemainsInspectable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte(`<Error><Code>EntityTooLarge</Code><Message>File exceeds storage limit</Message></Error>`))
	}))
	defer server.Close()
	_, err := uploadWithS3CompatibleBackend(uploadTestFile{bytes.NewReader([]byte("image"))}, "image.jpg", "image/jpeg", s3UploadTarget{
		bucket: "test-bucket", region: "local", endpoint: server.URL, accessKeyID: "fake", secretAccessKey: "fake",
	})
	if err == nil {
		t.Fatal("expected upload failure")
	}
	var apiError smithy.APIError
	if !errors.As(err, &apiError) || apiError.ErrorCode() != "EntityTooLarge" {
		t.Fatalf("underlying API error was lost: %v", err)
	}
	var responseError *smithyhttp.ResponseError
	if !errors.As(err, &responseError) || responseError.HTTPStatusCode() != 413 {
		t.Fatalf("underlying HTTP status was lost: %v", err)
	}
}
