package controllers

import (
	"ArmadaCMS/main/utils"
	"bytes"
	"errors"
	"mime/multipart"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestReadBlogpostHeaderImages(t *testing.T) {
	for _, test := range []struct {
		name      string
		manifest  string
		want      []string
		wantError bool
	}{
		{"omitted preserves existing", "", []string{"https://example.com/old.jpg"}, false},
		{"empty clears", "[]", []string{}, false},
		{"ordered uploads and URLs", `[{"file":"first"},{"url":" https://example.com/kept.jpg "},{"file":"second"}]`, []string{"https://example.com/first.jpg", "https://example.com/kept.jpg", "https://example.com/second.jpg"}, false},
		{"invalid JSON", "{", nil, true},
		{"null", "null", nil, true},
		{"missing upload", `[{"file":"missing"}]`, nil, true},
		{"invalid URL", `[{"url":"javascript:alert(1)"}]`, nil, true},
		{"empty row", `[{}]`, nil, true},
		{"ambiguous row", `[{"url":"https://example.com/photo.jpg","file":"first"}]`, nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			if test.manifest != "" {
				_ = writer.WriteField("headerImages", test.manifest)
			}
			for _, name := range []string{"first", "second"} {
				part, err := writer.CreateFormFile(name, name+".jpg")
				if err != nil {
					t.Fatal(err)
				}
				_, _ = part.Write([]byte("photo"))
			}
			_ = writer.Close()
			r := httptest.NewRequest("POST", "/", &body)
			r.Header.Set("Content-Type", writer.FormDataContentType())
			if err := r.ParseMultipartForm(1024); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := r.MultipartForm.RemoveAll(); err != nil {
					t.Errorf("Failed to remove multipart temporary files: %v", err)
				}
			})
			got, err := readBlogpostHeaderImages(r, []string{"https://example.com/old.jpg"}, func(_ multipart.File, header *multipart.FileHeader) (string, error) {
				return "https://example.com/" + header.Filename, nil
			})
			if test.wantError {
				if !errors.Is(err, errInvalidHeaderImages) {
					t.Fatalf("expected validation error, got %v", err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %v, %v; want %v", got, err, test.want)
			}
		})
	}
}

func TestWriteBlogpostImageError(t *testing.T) {
	for _, err := range []error{errInvalidHeaderImages, utils.ErrFileTooLarge, utils.ErrUnsupportedImageFormat, &blogpostImageFailure{Index: 1, Field: "headerImage1", Header: &multipart.FileHeader{Filename: "large.jpg"}, Stage: "upload", Err: utils.ErrFileTooLarge}} {
		w := httptest.NewRecorder()
		writeBlogpostImageError(w, err)
		if w.Code != 400 {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	}
}
