package controllers

import "net/http"

// Keep the existing body and status for API consumers; the UI translates the code.
func writePhotoPublicError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("X-Armada-Error-Code", code)
	http.Error(w, message, status)
}
