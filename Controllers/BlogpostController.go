package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// GetBlogposts returns a paginated list of blog posts.
// @Summary List blog posts
// @Tags blogposts
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"createdAt\",\"DESC\"]"
// @Param filter query string false "Filter"
// @Success 200 {array} models.Blogpost
// @Header 200 {string} Content-Range "blogposts 0-24/100"
// @Router /blogposts [get]
func GetBlogposts(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var items []models.Blogpost
	query := db.DB.Model(&models.Blogpost{})

	for k, v := range params.Filter {
		query = query.Where(utils.ToSnakeCase(k)+" = ?", v)
	}

	if len(params.Sort) == 2 {
		query = query.Order(utils.ToSnakeCase(params.Sort[0]) + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.Blogpost{}).Count(&total)

	query.Offset(start).Limit(limit).Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("blogposts %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetBlogpostByID returns a single blog post by ID.
// @Summary Get blog post by ID
// @Tags blogposts
// @Produce json
// @Param id path int true "Blogpost ID"
// @Success 200 {object} models.Blogpost
// @Failure 404 {string} string "Blogpost not found"
// @Router /blogposts/{id} [get]
func GetBlogpostByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var item models.Blogpost
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "Blogpost not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// CreateBlogpost creates a new blog post. Accepts multipart/form-data.
// @Summary Create blog post
// @Tags blogposts
// @Accept mpfd
// @Produce json
// @Param title formData string true "Title"
// @Param text formData string true "Markdown content"
// @Param author formData string true "Author name"
// @Param file formData file false "Cover image"
// @Success 201 {object} models.Blogpost
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /blogposts [post]
func CreateBlogpost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	var item models.Blogpost
	item.Title = r.FormValue("title")
	item.Text = r.FormValue("text")
	item.Author = r.FormValue("author")
	item.ShowCoverInPost = r.FormValue("showCoverInPost") != "false"

	imageUrl := r.FormValue("imageUrl")
	if imageUrl != "" {
		item.ImageURL = &imageUrl
	} else {
		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			fileURL, err := utils.UploadToS3(file, header)
			if err != nil {
				if errors.Is(err, utils.ErrUnsupportedImageFormat) {
					http.Error(w, "Unsupported image format. Allowed formats: JPG, JPEG, PNG, WEBP, GIF.", http.StatusBadRequest)
					return
				}
				if errors.Is(err, utils.ErrFileTooLarge) {
					http.Error(w, "Image file is too large. Maximum allowed size is 15 MB.", http.StatusBadRequest)
					return
				}
				http.Error(w, "Failed to upload image", http.StatusInternalServerError)
				return
			}
			item.ImageURL = &fileURL
		}
	}

	if err := createWithAudit(r, "blogposts", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, nil); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	writeCreatedJSONResponse(w, item)
}

// UpdateBlogpost updates a blog post by ID. Accepts multipart/form-data.
// @Summary Update blog post
// @Tags blogposts
// @Accept mpfd
// @Produce json
// @Param id path int true "Blogpost ID"
// @Param title formData string false "Title"
// @Param text formData string false "Markdown content"
// @Param author formData string false "Author name"
// @Param file formData file false "Cover image"
// @Success 200 {object} models.Blogpost
// @Failure 400 {string} string "Bad request"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /blogposts/{id} [put]
func UpdateBlogpost(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var item models.Blogpost
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "Blogpost not found", http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	updateMap := map[string]interface{}{
		"title":              r.FormValue("title"),
		"text":               r.FormValue("text"),
		"author":             r.FormValue("author"),
		"show_cover_in_post": r.FormValue("showCoverInPost") != "false",
	}

	imageUrl := r.FormValue("imageUrl")
	if imageUrl != "" {
		updateMap["image_url"] = imageUrl
	} else {
		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			fileURL, err := utils.UploadToS3(file, header)
			if err != nil {
				if errors.Is(err, utils.ErrUnsupportedImageFormat) {
					http.Error(w, "Unsupported image format. Allowed formats: JPG, JPEG, PNG, WEBP, GIF.", http.StatusBadRequest)
					return
				}
				if errors.Is(err, utils.ErrFileTooLarge) {
					http.Error(w, "Image file is too large. Maximum allowed size is 15 MB.", http.StatusBadRequest)
					return
				}
				http.Error(w, "Failed to upload image", http.StatusInternalServerError)
				return
			}
			updateMap["image_url"] = fileURL
		}
	}

	before := item
	if err := updateWithAudit(r, "blogposts", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updateMap).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&item, id).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// UploadBlogImage uploads an image for use in blog post markdown content.
// @Summary Upload blog inline image
// @Tags blogposts
// @Accept mpfd
// @Produce json
// @Param file formData file true "Image file"
// @Success 200 {object} map[string]string "url"
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Upload failed"
// @Security BearerAuth
// @Router /blogposts/upload [post]
func UploadBlogImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileURL, err := utils.UploadToS3(file, header)
	if err != nil {
		if errors.Is(err, utils.ErrUnsupportedImageFormat) {
			http.Error(w, "Unsupported image format. Allowed formats: JPG, JPEG, PNG, WEBP, GIF.", http.StatusBadRequest)
			return
		}
		if errors.Is(err, utils.ErrFileTooLarge) {
			http.Error(w, "Image file is too large. Maximum allowed size is 15 MB.", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": fileURL})
}

// DeleteBlogpost deletes a blog post by ID.
// @Summary Delete blog post
// @Tags blogposts
// @Produce json
// @Param id path int true "Blogpost ID"
// @Success 204 "Deleted"
// @Failure 404 {string} string "Blogpost not found"
// @Security BearerAuth
// @Router /blogposts/{id} [delete]
func DeleteBlogpost(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.Blogpost](w, r, "blogposts", id, "Blogpost not found", nil)
}
