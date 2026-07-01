package controllers

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func requestHasValidAccessToken(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return false
	}

	_, err := utils.VerifyAccessToken(parts[1])
	return err == nil
}

// allowedBlogpostColumns maps client-supplied field names to safe column names.
var allowedBlogpostColumns = map[string]string{
	"id":         "id",
	"title":      "title",
	"author":     "author",
	"published":  "published",
	"createdat":  "created_at",
	"created_at": "created_at",
}

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

	if !requestHasValidAccessToken(r) {
		query = query.Where("published = ?", true)
	}

	for k, v := range params.Filter {
		col, ok := allowedBlogpostColumns[strings.ToLower(k)]
		if !ok {
			continue
		}
		query = query.Where(col+" = ?", v)
	}

	if len(params.Sort) == 2 {
		col, ok := allowedBlogpostColumns[strings.ToLower(params.Sort[0])]
		dir := strings.ToUpper(params.Sort[1])
		if ok && (dir == "ASC" || dir == "DESC") {
			query = query.Order(col + " " + dir)
		}
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		http.Error(w, "Failed to count blogposts", http.StatusInternalServerError)
		return
	}

	if err := query.Offset(start).Limit(limit).Find(&items).Error; err != nil {
		http.Error(w, "Failed to fetch blogposts", http.StatusInternalServerError)
		return
	}

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

	if !item.Published && !requestHasValidAccessToken(r) {
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
	userID, ok := auth.GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	item.UserID = int64(userID)
	item.Title = r.FormValue("title")
	item.Text = r.FormValue("text")
	item.Author = r.FormValue("author")
	item.Published = r.FormValue("published") != "false"
	item.ShowCoverInPost = r.FormValue("showCoverInPost") != "false"

	imageUrl := r.FormValue("imageUrl")
	if imageUrl != "" {
		item.ImageURL = &imageUrl
	} else {
		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			fileURL, err := utils.UploadImage(file, header)
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
	}, nil, "blog-posts"); err != nil {
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
		"published":          r.FormValue("published") != "false",
		"show_cover_in_post": r.FormValue("showCoverInPost") != "false",
	}

	// A newly uploaded file takes priority over imageUrl.
	file, header, err := r.FormFile("file")
	if err == nil {
		defer file.Close()
		fileURL, err := utils.UploadImage(file, header)
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
	} else {
		// No new file — use the imageUrl field only if it was explicitly provided.
		if imageUrl := r.FormValue("imageUrl"); imageUrl != "" {
			updateMap["image_url"] = imageUrl
		}
	}

	before := item
	if err := updateWithAudit(r, "blogposts", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updateMap).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&item, id).Error
	}, "blog-posts"); err != nil {
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

	fileURL, err := utils.UploadImage(file, header)
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
	writeDeleteResponseWithAudit[models.Blogpost](w, r, "blogposts", id, "Blogpost not found", nil, nil, "blog-posts")
}
