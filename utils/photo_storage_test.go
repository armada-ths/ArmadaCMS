package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestSignPrivatePhotoUsesBrowserReachableEndpoint(t *testing.T) {
	t.Setenv("S3_ENDPOINT", "http://minio:9000")
	t.Setenv("PHOTO_S3_PRESIGN_ENDPOINT", "http://127.0.0.1:9000")
	t.Setenv("S3_BUCKET", "public-files")
	t.Setenv("PHOTO_S3_BUCKET", "event-photos")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	value, err := SignPrivatePhoto(context.Background(), "events/4/photo.jpg", false, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "127.0.0.1:9000" || parsed.Path != "/event-photos/events/4/photo.jpg" {
		t.Fatalf("signed URL uses wrong endpoint or bucket: %s%s", parsed.Host, parsed.Path)
	}
	if parsed.Query().Get("X-Amz-Signature") == "" {
		t.Fatal("signed URL is missing its signature")
	}
}

// Run against the local MinIO container with PHOTO_STORAGE_INTEGRATION=1.
func TestSignedPhotoURLCanBeReadFromMinIO(t *testing.T) {
	if os.Getenv("PHOTO_STORAGE_INTEGRATION") != "1" {
		t.Skip("requires local MinIO")
	}
	ctx := context.Background()
	key := fmt.Sprintf("integration/signed-photo-%d.jpg", time.Now().UnixNano())
	content := []byte("photo storage signing test")
	if err := UploadPrivatePhoto(ctx, key, bytes.NewReader(content), "image/jpeg", false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := DeletePrivatePhoto(ctx, key, false); err != nil {
			t.Error(err)
		}
	})
	signedURL, err := SignPrivatePhoto(ctx, key, false, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the signed browser-facing Host header but connect through Docker DNS.
	transport := &http.Transport{DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, "minio:9000")
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	response, err := client.Get(signedURL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("signed URL returned HTTP %d", response.StatusCode)
	}
	got, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatal("signed URL returned unexpected content")
	}
}
