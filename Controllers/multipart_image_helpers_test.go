package controllers

import (
	"ArmadaCMS/main/utils"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestApplyMultipartImageUpdate(t *testing.T) {
	for _, test := range []struct {
		name        string
		url         *string
		file        bool
		uploadError error
		want        map[string]any
	}{
		{name: "omitted preserves stored image", want: map[string]any{"title": "Entry"}},
		{name: "empty clears", url: stringPointer(""), want: map[string]any{"title": "Entry", "image_url": nil}},
		{name: "URL replaces", url: stringPointer("https://example.com/replacement.jpg"), want: map[string]any{"title": "Entry", "image_url": "https://example.com/replacement.jpg"}},
		{name: "upload replaces", file: true, want: map[string]any{"title": "Entry", "image_url": "https://example.com/uploaded.jpg"}},
		{name: "upload wins over retained URL", url: stringPointer("https://example.com/old.jpg"), file: true, want: map[string]any{"title": "Entry", "image_url": "https://example.com/uploaded.jpg"}},
		{name: "upload wins over empty URL", url: stringPointer(""), file: true, want: map[string]any{"title": "Entry", "image_url": "https://example.com/uploaded.jpg"}},
		{name: "failed upload does not fall back to retained URL", url: stringPointer("https://example.com/old.jpg"), file: true, uploadError: utils.ErrFileTooLarge, want: map[string]any{"title": "Entry"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			if test.url != nil {
				if err := writer.WriteField("imageUrl", *test.url); err != nil {
					t.Fatal(err)
				}
			}
			if test.file {
				part, err := writer.CreateFormFile("file", "replacement.jpg")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := part.Write([]byte("photo")); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPut, "/", &body)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			if err := request.ParseMultipartForm(1024); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := request.MultipartForm.RemoveAll(); err != nil {
					t.Errorf("multipart cleanup: %v", err)
				}
			})
			updates := map[string]any{"title": "Entry"}
			calls := 0
			header, err := applyMultipartImageUpdate(request, updates, func(file multipart.File, header *multipart.FileHeader) (string, error) {
				calls++
				content, err := io.ReadAll(file)
				if err != nil || string(content) != "photo" || header.Filename != "replacement.jpg" {
					t.Fatalf("unexpected upload: content=%q header=%+v err=%v", content, header, err)
				}
				return "https://example.com/uploaded.jpg", test.uploadError
			})
			if !errors.Is(err, test.uploadError) {
				t.Fatalf("got error %v, want %v", err, test.uploadError)
			}
			if !reflect.DeepEqual(updates, test.want) {
				t.Fatalf("got updates %#v, want %#v", updates, test.want)
			}
			if test.file {
				if calls != 1 || header == nil || header.Filename != "replacement.jpg" {
					t.Fatalf("upload calls=%d, header=%+v", calls, header)
				}
			} else if calls != 0 || header != nil {
				t.Fatalf("unexpected upload calls=%d header=%+v", calls, header)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }

func TestWriteImageUploadError(t *testing.T) {
	for _, test := range []struct {
		err     error
		status  int
		message string
	}{
		{fmt.Errorf("wrapped: %w", utils.ErrUnsupportedImageFormat), http.StatusBadRequest, "Unsupported image format"},
		{utils.ErrFileTooLarge, http.StatusBadRequest, "Maximum allowed size is 15 MB"},
		{errors.New("storage failure"), http.StatusInternalServerError, "Failed to upload image"},
	} {
		response := httptest.NewRecorder()
		writeImageUploadError(response, test.err)
		if response.Code != test.status || !strings.Contains(response.Body.String(), test.message) {
			t.Fatalf("got %d %q, want %d containing %q", response.Code, response.Body.String(), test.status, test.message)
		}
	}
}
