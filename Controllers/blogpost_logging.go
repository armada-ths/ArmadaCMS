package controllers

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

var blogpostFailureLogger = slog.New(slog.NewJSONHandler(os.Stderr, nil))

// Log server-side diagnostics without recording credentials or form contents.
func logBlogpostFailure(r *http.Request, stage string, err error, header *multipart.FileHeader) {
	attrs := []any{
		"severity", "ERROR", "stage", stage, "method", r.Method,
		"path", r.URL.Path, "request_size_bytes", r.ContentLength, "error", err.Error(),
	}
	traceID := strings.SplitN(r.Header.Get("X-Cloud-Trace-Context"), "/", 2)[0]
	if decoded, decodeErr := hex.DecodeString(traceID); decodeErr == nil && len(decoded) == 16 {
		attrs = append(attrs, "trace_id", traceID)
	}
	if r.MultipartForm != nil {
		fileCount := 0
		var uploadBytes int64
		for _, files := range r.MultipartForm.File {
			fileCount += len(files)
			for _, file := range files {
				uploadBytes += file.Size
			}
		}
		attrs = append(attrs, "upload_file_count", fileCount, "upload_total_bytes", uploadBytes)
	}
	var imageFailure *blogpostImageFailure
	if errors.As(err, &imageFailure) {
		header = imageFailure.Header
		attrs = append(attrs, "header_image_index", imageFailure.Index, "file_field", imageFailure.Field, "image_stage", imageFailure.Stage)
	}
	if header != nil {
		attrs = append(attrs, "filename", header.Filename, "file_size_bytes", header.Size)
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		attrs = append(attrs, "storage_error_code", apiError.ErrorCode(), "storage_error_message", apiError.ErrorMessage())
	}
	var responseError *smithyhttp.ResponseError
	if errors.As(err, &responseError) {
		attrs = append(attrs, "storage_http_status", responseError.HTTPStatusCode())
	}
	blogpostFailureLogger.Error("Blogpost request failed", attrs...)
}

// Preserve the underlying SDK error while identifying the failed gallery image.
type blogpostImageFailure struct {
	Index  int
	Field  string
	Header *multipart.FileHeader
	Stage  string
	Err    error
}

func (e *blogpostImageFailure) Error() string {
	return fmt.Sprintf("header image %d (%s, %q, %d bytes): %v", e.Index, e.Stage, e.Header.Filename, e.Header.Size, e.Err)
}

func (e *blogpostImageFailure) Unwrap() error { return e.Err }
