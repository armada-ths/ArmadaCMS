package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
)

const MaxPhotoOriginalBytes = 25 * 1024 * 1024
const MaxPhotoPixels = 60_000_000

var photoTransform = make(chan struct{}, 1)
var photoVipsOnce sync.Once

func DetectPhotoFormat(data []byte) (string, error) {
	if len(data) < 12 {
		return "", ErrUnsupportedImageFormat
	}
	contentType := http.DetectContentType(data[:min(len(data), 512)])
	if contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp" {
		return contentType, nil
	}
	if string(data[4:8]) == "ftyp" {
		brand := string(data[8:12])
		if strings.HasPrefix(brand, "hei") || strings.HasPrefix(brand, "hev") || strings.HasPrefix(brand, "mif") || strings.HasPrefix(brand, "msf") {
			return "image/heic", nil
		}
	}
	return "", ErrUnsupportedImageFormat
}

// ProcessPhoto stores no original: it exports only a metadata-free, bounded JPEG.
func ProcessPhoto(file multipart.File, size int64) ([]byte, int, int, error) {
	if size < 1 || size > MaxPhotoOriginalBytes {
		return nil, 0, 0, ErrFileTooLarge
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, 0, 0, errors.New("could not read original image")
	}
	if _, err := DetectPhotoFormat(data); err != nil {
		return nil, 0, 0, err
	}
	photoTransform <- struct{}{}
	defer func() { <-photoTransform }()
	photoVipsOnce.Do(func() { vips.Startup(nil) })
	image, err := vips.NewImageFromBuffer(data)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode image: %w", err)
	}
	defer image.Close()
	if image.Width() < 1 || image.Height() < 1 || int64(image.Width())*int64(image.Height()) > MaxPhotoPixels {
		return nil, 0, 0, errors.New("image dimensions exceed limit")
	}
	if err := image.AutoRotate(); err != nil {
		return nil, 0, 0, err
	}
	if err := image.OptimizeICCProfile(); err != nil {
		return nil, 0, 0, err
	}
	longest := max(image.Width(), image.Height())
	if longest > 2560 {
		if err := image.Resize(float64(2560)/float64(longest), vips.KernelLanczos3); err != nil {
			return nil, 0, 0, err
		}
	}
	if err := image.ToColorSpace(vips.InterpretationSRGB); err != nil {
		return nil, 0, 0, err
	}
	output, _, err := image.ExportJpeg(&vips.JpegExportParams{Quality: 88, StripMetadata: true})
	if err != nil {
		return nil, 0, 0, err
	}
	if len(output) > MaxPhotoOriginalBytes {
		return nil, 0, 0, ErrFileTooLarge
	}
	return bytes.Clone(output), image.Width(), image.Height(), nil
}
