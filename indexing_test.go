package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNoIndexPreservesResponses(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusPermanentRedirect, http.StatusUnauthorized, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			handler := noIndex(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = w.Write([]byte("response"))
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
			result := response.Result()
			defer result.Body.Close()
			if got := result.Header.Get("X-Robots-Tag"); got != "noindex, nofollow" {
				t.Fatalf("X-Robots-Tag = %q", got)
			}
			if result.StatusCode != status || response.Body.String() != "response" || result.Header.Get("Content-Type") != "application/json" {
				t.Fatal("indexing middleware changed the response")
			}
		})
	}
}
