package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPhotoPublicErrorKeepsStatusAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	writePhotoPublicError(w, "photo_limit_reached", "Photo limit reached", http.StatusTooManyRequests)
	if w.Code != 429 || w.Body.String() != "Photo limit reached\n" || w.Header().Get("X-Armada-Error-Code") != "photo_limit_reached" {
		t.Fatalf("%#v %s", w.Result(), w.Body.String())
	}
}
