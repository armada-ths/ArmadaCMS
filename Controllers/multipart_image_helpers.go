package controllers

import (
	"ArmadaCMS/main/utils"
	"errors"
	"mime/multipart"
	"net/http"
)

// applyMultipartImageUpdate reads a parsed multipart request. A new file wins
// over imageUrl; an omitted URL preserves the stored image, and an empty URL clears it.
// The header is returned for callers that log upload failures with file metadata.
func applyMultipartImageUpdate(r *http.Request, updates map[string]any, upload func(multipart.File, *multipart.FileHeader) (string, error)) (*multipart.FileHeader, error) {
	file, header, err := r.FormFile("file")
	if err == nil {
		defer file.Close()
		imageURL, err := upload(file, header)
		if err != nil {
			return header, err
		}
		updates["image_url"] = imageURL
		return header, nil
	}
	if r.MultipartForm != nil {
		if _, present := r.MultipartForm.Value["imageUrl"]; present {
			if imageURL := r.FormValue("imageUrl"); imageURL != "" {
				updates["image_url"] = imageURL
			} else {
				updates["image_url"] = nil
			}
		}
	}
	return nil, nil
}

func writeImageUploadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, utils.ErrUnsupportedImageFormat):
		http.Error(w, "Unsupported image format. Allowed formats: JPG, JPEG, PNG, WEBP, GIF.", http.StatusBadRequest)
	case errors.Is(err, utils.ErrFileTooLarge):
		http.Error(w, "Image file is too large. Maximum allowed size is 15 MB.", http.StatusBadRequest)
	default:
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
	}
}
