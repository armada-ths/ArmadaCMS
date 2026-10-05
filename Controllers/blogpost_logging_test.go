package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

func captureBlogpostLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	previous := blogpostFailureLogger
	blogpostFailureLogger = slog.New(slog.NewJSONHandler(&output, nil))
	t.Cleanup(func() { blogpostFailureLogger = previous })
	return &output
}

func TestLogBlogpostStorageFailure(t *testing.T) {
	output := captureBlogpostLog(t)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/blogposts/24?secret=query-secret", nil)
	request.ContentLength = 19443115
	request.Header.Set("Authorization", "Bearer auth-secret")
	request.Header.Set("Cookie", "session=cookie-secret")
	request.Header.Set("X-Cloud-Trace-Context", "2634ed1cf56bdeaaa5bc988b9d276851/123;o=1")
	header := &multipart.FileHeader{Filename: "P1088087.JPG", Size: 6291456}
	request.MultipartForm = &multipart.Form{
		File:  map[string][]*multipart.FileHeader{"headerImage2": {header}},
		Value: map[string][]string{"text": {"private-blog-text"}},
	}
	apiError := &smithy.GenericAPIError{Code: "EntityTooLarge", Message: "upload exceeds limit"}
	responseError := &smithyhttp.ResponseError{Response: &smithyhttp.Response{Response: &http.Response{StatusCode: 413}}, Err: apiError}
	failure := &blogpostImageFailure{Index: 2, Field: "headerImage2", Header: header, Stage: "upload", Err: fmt.Errorf("upload failed: %w", responseError)}
	logBlogpostFailure(request, "upload_header_images", failure, nil)
	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"severity": "ERROR", "stage": "upload_header_images", "path": "/api/v1/blogposts/24",
		"trace_id": "2634ed1cf56bdeaaa5bc988b9d276851", "filename": "P1088087.JPG",
		"header_image_index": float64(2), "file_field": "headerImage2", "image_stage": "upload",
		"file_size_bytes": float64(6291456), "request_size_bytes": float64(19443115),
		"upload_file_count": float64(1), "upload_total_bytes": float64(6291456),
		"storage_error_code": "EntityTooLarge", "storage_http_status": float64(413),
	} {
		if record[key] != want {
			t.Errorf("%s = %v; want %v", key, record[key], want)
		}
	}
	for _, secret := range []string{"query-secret", "auth-secret", "cookie-secret", "private-blog-text"} {
		if strings.Contains(output.String(), secret) {
			t.Errorf("log contains %s", secret)
		}
	}
	var recovered smithy.APIError
	if !errors.As(failure, &recovered) || recovered != apiError {
		t.Fatal("underlying API error was lost")
	}
}

func TestLogBlogpostDatabaseFailure(t *testing.T) {
	output := captureBlogpostLog(t)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/blogposts/24", nil)
	request.Header.Set("X-Cloud-Trace-Context", "invalid-untrusted-header")
	logBlogpostFailure(request, "update_database_audit", errors.New("transaction failed"), nil)
	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["error"] != "transaction failed" {
		t.Fatal("missing database error")
	}
	for _, field := range []string{"trace_id", "filename", "storage_http_status"} {
		if _, present := record[field]; present {
			t.Errorf("unexpected field %s", field)
		}
	}
}
