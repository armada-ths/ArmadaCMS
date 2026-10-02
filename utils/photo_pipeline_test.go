package utils

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

type memoryPhoto struct{ *bytes.Reader }

func (*memoryPhoto) Close() error { return nil }

func TestPhotoPipelineAcceptsJPEGAndPNG(t *testing.T) {
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
	original := jpegBuffer.Bytes()
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
	if pngOutput, _, _, err := ProcessPhoto(&memoryPhoto{bytes.NewReader(pngBuffer.Bytes())}, int64(pngBuffer.Len())); err != nil || len(pngOutput) < 2 || pngOutput[0] != 0xff || pngOutput[1] != 0xd8 {
		t.Fatalf("PNG was not converted to JPEG: %v", err)
	}
	corruptJPEG := []byte{0xff, 0xd8, 0xff, 0xe0}
	if _, _, _, err := ProcessPhoto(&memoryPhoto{bytes.NewReader(corruptJPEG)}, int64(len(corruptJPEG))); err == nil {
		t.Fatal("corrupt JPEG accepted")
	}
}
