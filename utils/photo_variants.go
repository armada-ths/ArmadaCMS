package utils

import (
	"ArmadaCMS/main/models"
	"bytes"
	"context"
	"time"
)

// StorePhotoThumbnail uses a versioned key so retries replace the same derivative.
func StorePhotoThumbnail(ctx context.Context, originalKey string, jpeg []byte) (string, error) {
	thumbnail, err := PhotoThumbnail(jpeg)
	if err != nil {
		return "", err
	}
	key := originalKey + ".thumb-v1.jpg"
	if err := UploadPrivatePhoto(ctx, key, bytes.NewReader(thumbnail), "image/jpeg", false); err != nil {
		return "", err
	}
	return key, nil
}
func PhotoThumbnailKey(photo models.EventPhoto) *string {
	if photo.ObjectKey == nil {
		return nil
	}
	if photo.ThumbnailObjectKey != nil {
		return photo.ThumbnailObjectKey
	}
	return photo.ObjectKey
}
func SignPhotoThumbnail(ctx context.Context, photo models.EventPhoto, ttl time.Duration) (string, error) {
	key := PhotoThumbnailKey(photo)
	if key == nil {
		return "", nil
	}
	return SignPrivatePhoto(ctx, *key, false, ttl)
}

// Call while holding the photo row lock, also used by thumbnail backfill.
func DeletePhotoVariants(ctx context.Context, photo models.EventPhoto) error {
	thumbnailKey := photo.ThumbnailObjectKey
	if thumbnailKey == nil && photo.ObjectKey != nil {
		key := *photo.ObjectKey + ".thumb-v1.jpg"
		thumbnailKey = &key
	}
	if thumbnailKey != nil {
		if err := DeletePrivatePhoto(ctx, *thumbnailKey, false); err != nil {
			return err
		}
	}
	if photo.ObjectKey != nil {
		return DeletePrivatePhoto(ctx, *photo.ObjectKey, false)
	}
	return nil
}
