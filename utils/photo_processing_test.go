package utils

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestDetectPhotoFormat(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"jpeg", append([]byte{0xff, 0xd8, 0xff, 0xe0}, make([]byte, 12)...), "image/jpeg"},
		{"png", append([]byte{137, 80, 78, 71, 13, 10, 26, 10}, make([]byte, 12)...), "image/png"},
		{"webp", []byte("RIFF\x10\x00\x00\x00WEBPVP8 "), "image/webp"},
		{"heic", []byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1heic"), "image/heic"},
		{"heif compatible brand", []byte("\x00\x00\x00\x18ftypmif1\x00\x00\x00\x00heicmif1"), "image/heic"},
		{"avif is not heic", []byte("\x00\x00\x00\x18ftypavif\x00\x00\x00\x00mif1avif"), ""},
		{"short jpeg", []byte{0xff, 0xd8}, ""},
		{"invalid", []byte("not an image at all"), ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DetectPhotoFormat(test.data)
			if test.want == "" {
				if err == nil {
					t.Fatal("invalid input accepted")
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("format: got %q, err %v", got, err)
			}
		})
	}
}

func TestPhotoSafeSearchImage(t *testing.T) {
	original := image.NewRGBA(image.Rect(0, 0, 1200, 800))
	original.Set(0, 0, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, original, nil); err != nil {
		t.Fatal(err)
	}
	analysisImage, err := PhotoSafeSearchImage(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(analysisImage))
	if err != nil {
		t.Fatal(err)
	}
	if max(config.Width, config.Height) > 640 || len(analysisImage) > 1024*1024 {
		t.Fatalf("analysis image is too large: %dx%d, %d bytes", config.Width, config.Height, len(analysisImage))
	}
}
