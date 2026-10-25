package main

import "net/http"

// noIndex covers admin pages, API responses, assets and errors in every environment.
// Crawling must remain allowed so search engines can read this directive.
func noIndex(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		next.ServeHTTP(w, r)
	})
}
