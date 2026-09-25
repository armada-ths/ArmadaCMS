package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
)

const MaxPhotoOriginalBytes = 25 * 1024 * 1024
const MaxPhotoPixels = 60_000_000

var photoTransform = make(chan struct{}, 1)
var photoVipsOnce sync.Once
var photoVipsStartupErr error

func startPhotoVips() error {
	photoVipsOnce.Do(func() { photoVipsStartupErr = vips.Startup(nil) })
	return photoVipsStartupErr
}

func DetectPhotoFormat(data []byte) (string, error) {
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "image/jpeg", nil
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
	if err := startPhotoVips(); err != nil {
		return nil, 0, 0, fmt.Errorf("start image processor: %w", err)
	}
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
