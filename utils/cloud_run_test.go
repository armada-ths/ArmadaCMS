package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type cloudRunTransport func(*http.Request) (*http.Response, error)

func (f cloudRunTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type cloudRunBody struct {
	io.Reader
	closed bool
}

func (b *cloudRunBody) Close() error { b.closed = true; return nil }

func TestStartCloudRunJob(t *testing.T) {
	for _, status := range []int{200, 202, 204, 299, 300, 401, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			body := &cloudRunBody{Reader: strings.NewReader("response")}
			called := false
			client := &http.Client{Transport: cloudRunTransport(func(r *http.Request) (*http.Response, error) {
				called = true
				if r.Context() != ctx || r.Method != http.MethodPost || r.URL.String() != "https://run.googleapis.com/v2/projects/test/locations/europe-north2/jobs/worker:run" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Content-Type") != "application/json" {
					t.Fatalf("unexpected headers: %v", r.Header)
				}
				payload, err := io.ReadAll(r.Body)
				if err != nil || string(payload) != "{}" {
					t.Fatalf("unexpected payload: %q, %v", payload, err)
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: body}, nil
			})}
			err := startCloudRunJob(ctx, "  projects/test/locations/europe-north2/jobs/worker  ", func(tokenCtx context.Context) (string, error) {
				if tokenCtx != ctx {
					t.Fatal("token request lost context")
				}
				return "test-token", nil
			}, client)
			if !called || !body.closed {
				t.Fatal("request not sent or response body not closed")
			}
			if status < 300 && err != nil {
				t.Fatal(err)
			}
			if status >= 300 && (err == nil || !strings.Contains(err.Error(), fmt.Sprint(status))) {
				t.Fatalf("missing status error: %v", err)
			}
		})
	}
}

func TestStartCloudRunJobFailures(t *testing.T) {
	sentinel := errors.New("token unavailable")
	client := &http.Client{Transport: cloudRunTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("must not send request")
		return nil, nil
	})}
	if err := startCloudRunJob(context.Background(), " ", func(context.Context) (string, error) {
		t.Fatal("must not fetch token for missing job")
		return "", nil
	}, client); err == nil {
		t.Fatal("missing job name accepted")
	}
	if err := startCloudRunJob(context.Background(), "projects/test/locations/europe-north2/jobs/worker", func(context.Context) (string, error) {
		return "", sentinel
	}, client); !errors.Is(err, sentinel) {
		t.Fatalf("token error lost: %v", err)
	}
	client.Transport = cloudRunTransport(func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })
	if err := startCloudRunJob(context.Background(), "projects/test/locations/europe-north2/jobs/worker", func(context.Context) (string, error) {
		return "test-token", nil
	}, client); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("request error lost: %v", err)
	}
}
