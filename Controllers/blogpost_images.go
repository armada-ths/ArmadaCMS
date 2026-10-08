package controllers

import (
	"ArmadaCMS/main/utils"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

var errInvalidHeaderImages = errors.New("invalid header images")

type blogpostHeaderImage struct {
	URL  string `json:"url"`
	File string `json:"file"`
}

// An omitted manifest preserves existing photos; an empty array removes them.
// File references allow retained URLs and new uploads to be interleaved in order.
func readBlogpostHeaderImages(r *http.Request, existing []string, upload func(multipart.File, *multipart.FileHeader) (string, error)) ([]string, error) {
	values, present := r.MultipartForm.Value["headerImages"]
	if !present {
		return append([]string{}, existing...), nil
	}
	var entries []blogpostHeaderImage
	if len(values) != 1 || json.Unmarshal([]byte(values[0]), &entries) != nil || entries == nil {
		return nil, fmt.Errorf("%w: expected a JSON array", errInvalidHeaderImages)
	}
	// Validate the full manifest before uploading any files.
	for i := range entries {
		entry := &entries[i]
		entry.URL = strings.TrimSpace(entry.URL)
		if (entry.URL == "") == (entry.File == "") {
			return nil, fmt.Errorf("%w: each image needs either a URL or a file", errInvalidHeaderImages)
		}
		if entry.File != "" {
			if len(r.MultipartForm.File[entry.File]) != 1 {
				return nil, fmt.Errorf("%w: missing image file", errInvalidHeaderImages)
			}
		} else {
			parsed, err := url.Parse(entry.URL)
			if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return nil, fmt.Errorf("%w: image URLs must use http or https", errInvalidHeaderImages)
			}
		}
	}
	images := make([]string, 0, len(entries))
	for index, entry := range entries {
		if entry.File == "" {
			images = append(images, entry.URL)
			continue
		}
		header := r.MultipartForm.File[entry.File][0]
		file, err := header.Open()
		if err != nil {
			return nil, &blogpostImageFailure{Index: index + 1, Field: entry.File, Header: header, Stage: "open", Err: err}
		}
		imageURL, err := upload(file, header)
		file.Close()
		if err != nil {
			return nil, &blogpostImageFailure{Index: index + 1, Field: entry.File, Header: header, Stage: "upload", Err: err}
		}
		images = append(images, imageURL)
	}
	return images, nil
}

func writeBlogpostImageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalidHeaderImages):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, utils.ErrUnsupportedImageFormat):
		http.Error(w, "Unsupported image format. Allowed formats: JPG, JPEG, PNG, WEBP, GIF.", http.StatusBadRequest)
	case errors.Is(err, utils.ErrFileTooLarge):
		http.Error(w, "Image file is too large. Maximum allowed size is 15 MB.", http.StatusBadRequest)
	default:
		http.Error(w, "Failed to upload header image", http.StatusInternalServerError)
	}
}
