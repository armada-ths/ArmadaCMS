package utils

import (
	"ArmadaCMS/main/models"
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestPhotoThumbnail(t *testing.T) {
	for _, size := range []image.Point{{2560, 1700}, {1000, 2400}, {80, 40}} {
		source := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
		for y := 0; y < size.Y; y++ {
			for x := 0; x < size.X; x++ {
				source.SetRGBA(x, y, color.RGBA{uint8(x*31 + y*7), uint8(y*29 + x*3), uint8(x + y), 255})
			}
		}
		var original bytes.Buffer
		if err := jpeg.Encode(&original, source, &jpeg.Options{Quality: 88}); err != nil {
			t.Fatal(err)
		}
		before := bytes.Clone(original.Bytes())
		thumbnail, err := PhotoThumbnail(original.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(thumbnail))
		if err != nil {
			t.Fatal(err)
		}
		if max(cfg.Width, cfg.Height) > 640 || cfg.Width > size.X || cfg.Height > size.Y {
			t.Fatalf("unexpected dimensions: %+v", cfg)
		}
		if abs(cfg.Width*size.Y-cfg.Height*size.X) > max(size.X, size.Y) {
			t.Fatal("aspect ratio changed")
		}
		if !bytes.Equal(before, original.Bytes()) {
			t.Fatal("full-size image was changed")
		}
		if bytes.Contains(thumbnail, []byte("Exif")) {
			t.Fatal("EXIF retained")
		}
		if size.X > 640 && len(thumbnail) >= len(before)/5 {
			t.Fatalf("thumbnail reduction below 80 percent: %d/%d", len(thumbnail), len(before))
		}
	}
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func TestPhotoThumbnailOrientation(t *testing.T) {
	var original bytes.Buffer
	_ = jpeg.Encode(&original, image.NewRGBA(image.Rect(0, 0, 1200, 800)), nil)
	// APP1 EXIF orientation 6 (rotate 90 degrees).
	exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	length := len(exif) + 2
	encoded := append([]byte{255, 216, 255, 225, byte(length >> 8), byte(length)}, exif...)
	encoded = append(encoded, original.Bytes()[2:]...)
	thumbnail, err := PhotoThumbnail(encoded)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(thumbnail))
	if err != nil || cfg.Height != 640 || cfg.Width != 427 {
		t.Fatalf("orientation: %+v, %v", cfg, err)
	}
	if bytes.Contains(thumbnail, []byte("Exif")) {
		t.Fatal("EXIF retained")
	}
}
func TestPhotoThumbnailKeyFallback(t *testing.T) {
	full, thumb := "full.jpg", "thumb.jpg"
	if PhotoThumbnailKey(models.EventPhoto{ThumbnailObjectKey: &thumb}) != nil {
		t.Fatal("deleted photo exposed thumbnail")
	}
	if got := PhotoThumbnailKey(models.EventPhoto{ObjectKey: &full}); got == nil || *got != full {
		t.Fatal("missing fallback")
	}
	if got := PhotoThumbnailKey(models.EventPhoto{ObjectKey: &full, ThumbnailObjectKey: &thumb}); got == nil || *got != thumb {
		t.Fatal("not using thumbnail")
	}
}
