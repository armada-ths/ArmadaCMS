package utils

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

type memoryPhoto struct{ *bytes.Reader }

func (*memoryPhoto) Close() error { return nil }

func TestPhotoPipelineConvertsSupportedFormats(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 12, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 12; x++ {
			source.Set(x, y, color.RGBA{R: 180, G: 80, B: 100, A: 255})
		}
	}
	var pngBuffer, jpegBuffer bytes.Buffer
	if err := png.Encode(&pngBuffer, source); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&jpegBuffer, source, nil); err != nil {
		t.Fatal(err)
	}
	inputs := map[string][]byte{"png": pngBuffer.Bytes(), "jpeg": jpegBuffer.Bytes()}
	photoVipsOnce.Do(func() { vips.Startup(nil) })
	ref, err := vips.NewImageFromBuffer(pngBuffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer ref.Close()
	if webp, _, err := ref.ExportWebp(vips.NewWebpExportParams()); err == nil {
		inputs["webp"] = webp
	} else {
		t.Fatal(err)
	}
	if heic, _, err := ref.ExportHeif(vips.NewHeifExportParams()); err == nil {
		inputs["heic"] = heic
	} else {
		t.Fatal(err)
	}
	for name, original := range inputs {
		t.Run(name, func(t *testing.T) {
			output, width, height, err := ProcessPhoto(&memoryPhoto{bytes.NewReader(original)}, int64(len(original)))
			if err != nil {
				t.Fatal(err)
			}
			if width != 12 || height != 8 {
				t.Fatalf("dimensions %dx%d", width, height)
			}
			if len(output) < 2 || output[0] != 0xff || output[1] != 0xd8 {
				t.Fatal("output is not JPEG")
			}
		})
	}
}
