package utils

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type uploadTestFile struct{ *bytes.Reader }

func (uploadTestFile) Close() error { return nil }

func TestUploadStorageErrorRemainsInspectable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte(`<Error><Code>EntityTooLarge</Code><Message>File exceeds storage limit</Message></Error>`))
	}))
	defer server.Close()
	_, err := uploadWithS3CompatibleBackend(uploadTestFile{bytes.NewReader([]byte("image"))}, "image.jpg", "image/jpeg", s3UploadTarget{
		bucket: "test-bucket", region: "local", endpoint: server.URL, accessKeyID: "fake", secretAccessKey: "fake",
	})
	if err == nil {
		t.Fatal("expected upload failure")
	}
	var apiError smithy.APIError
	if !errors.As(err, &apiError) || apiError.ErrorCode() != "EntityTooLarge" {
		t.Fatalf("underlying API error was lost: %v", err)
	}
	var responseError *smithyhttp.ResponseError
	if !errors.As(err, &responseError) || responseError.HTTPStatusCode() != 413 {
		t.Fatalf("underlying HTTP status was lost: %v", err)
	}
}

// Exercise the real HTTPS checksum/trailer path. Plain HTTP doesn't use
// aws-chunked encoding and therefore wouldn't reproduce this Storage failure.
func TestImageUploadHTTPSChunkLimit(t *testing.T) {
	for _, size := range []int{(8 << 20) - 1, 8 << 20, 8652288, 15 << 20} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			payload := bytes.Repeat([]byte("camera-image-data"), (size/17)+1)[:size]
			var mu sync.Mutex
			parts := map[int][]byte{}
			var single []byte
			completed := false
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				query := r.URL.Query()
				switch {
				case r.Method == http.MethodPost && query.Has("uploads"):
					_, _ = io.WriteString(w, `<InitiateMultipartUploadResult><Bucket>test-bucket</Bucket><Key>image.jpg</Key><UploadId>test-upload</UploadId></InitiateMultipartUploadResult>`)
				case r.Method == http.MethodPut:
					if r.Header.Get("Content-Encoding") != "aws-chunked" {
						t.Error("test did not exercise HTTPS aws-chunked encoding")
					}
					data, err := readTestAWSChunks(r.Body, 8<<20)
					if err != nil {
						w.WriteHeader(http.StatusRequestEntityTooLarge)
						_, _ = fmt.Fprintf(w, `<Error><Code>EntityTooLarge</Code><Message>%s</Message></Error>`, err)
						return
					}
					mu.Lock()
					defer mu.Unlock()
					if query.Get("uploadId") != "" {
						partNumber, parseErr := strconv.Atoi(query.Get("partNumber"))
						if parseErr != nil {
							t.Error(parseErr)
							w.WriteHeader(400)
							return
						}
						parts[partNumber] = data
					} else {
						single = data
					}
					w.Header().Set("ETag", `"test-etag"`)
				case r.Method == http.MethodPost && query.Get("uploadId") != "":
					mu.Lock()
					completed = true
					mu.Unlock()
					_, _ = io.WriteString(w, `<CompleteMultipartUploadResult><Bucket>test-bucket</Bucket><Key>image.jpg</Key><ETag>"test-etag"</ETag></CompleteMultipartUploadResult>`)
				default:
					t.Errorf("unexpected S3 request %s %s", r.Method, r.URL.RequestURI())
					w.WriteHeader(400)
				}
			}))
			defer server.Close()
			client := s3.NewFromConfig(aws.Config{
				Region: "local", Credentials: credentials.NewStaticCredentialsProvider("fake", "fake", ""), HTTPClient: server.Client(),
			}, func(o *s3.Options) { o.BaseEndpoint = aws.String(server.URL); o.UsePathStyle = true })
			_, err := newImageUploader(client).UploadObject(context.Background(), &transfermanager.UploadObjectInput{
				Bucket: aws.String("test-bucket"), Key: aws.String("image.jpg"), Body: bytes.NewReader(payload), ContentType: aws.String("image/jpeg"),
			})
			if err != nil {
				t.Fatalf("upload of %d bytes failed: %v", size, err)
			}
			mu.Lock()
			defer mu.Unlock()
			var uploaded []byte
			if size < 8<<20 {
				uploaded = single
				if completed || len(parts) != 0 {
					t.Error("small image unexpectedly used multipart")
				}
			} else {
				if !completed {
					t.Fatal("large image wasn't completed via multipart")
				}
				for partNumber := 1; partNumber <= len(parts); partNumber++ {
					part := parts[partNumber]
					if len(part) > 8<<20 {
						t.Errorf("part %d exceeds chunk limit", partNumber)
					}
					if partNumber < len(parts) && len(part) < 5<<20 {
						t.Errorf("non-final part %d is smaller than S3 minimum", partNumber)
					}
					uploaded = append(uploaded, part...)
				}
			}
			if !bytes.Equal(uploaded, payload) {
				t.Fatal("uploaded image differs from original")
			}
		})
	}
}

// Mimic Storage's aws-chunked parser, enforcing its per-chunk bound before
// consuming the payload. HTTP transfer framing is already decoded by net/http.
func readTestAWSChunks(body io.Reader, maxChunk int64) ([]byte, error) {
	reader := bufio.NewReader(body)
	var payload bytes.Buffer
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		chunkSize, err := strconv.ParseInt(strings.SplitN(strings.TrimSpace(line), ";", 2)[0], 16, 64)
		if err != nil {
			return nil, err
		}
		if chunkSize > maxChunk {
			return nil, fmt.Errorf("The chunk exceeded %d bytes", maxChunk)
		}
		if chunkSize < 0 {
			return nil, fmt.Errorf("negative chunk size")
		}
		if chunkSize == 0 {
			if _, err := io.Copy(io.Discard, reader); err != nil {
				return nil, err
			}
			return payload.Bytes(), nil
		}
		if _, err := io.CopyN(&payload, reader, chunkSize); err != nil {
			return nil, err
		}
		var separator [2]byte
		if _, err := io.ReadFull(reader, separator[:]); err != nil {
			return nil, err
		}
		if string(separator[:]) != "\r\n" {
			return nil, fmt.Errorf("invalid chunk delimiter")
		}
	}
}
