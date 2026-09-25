package utils

import "testing"

func TestDetectPhotoFormat(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"jpeg", append([]byte{0xff, 0xd8, 0xff, 0xe0}, make([]byte, 12)...), "image/jpeg"},
		{"png", append([]byte{137, 80, 78, 71, 13, 10, 26, 10}, make([]byte, 12)...), ""},
		{"webp", []byte("RIFF\x10\x00\x00\x00WEBPVP8 "), ""},
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
