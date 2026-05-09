package utils

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

var revalidateClient = &http.Client{Timeout: 5 * time.Second}

// RevalidateTag sends a fire-and-forget POST to the public site's on-demand
// revalidation endpoint, causing Next.js to purge its cache for the given tag.
// If REVALIDATION_URL or REVALIDATION_SECRET is not set, the call is silently skipped.
func RevalidateTag(tag string) {
	url := os.Getenv("REVALIDATION_URL")
	secret := os.Getenv("REVALIDATION_SECRET")
	if url == "" || secret == "" {
		return
	}

	body, err := json.Marshal(map[string]string{"tag": tag, "secret": secret})
	if err != nil {
		log.Printf("revalidation: failed to marshal payload for tag %q: %v", tag, err)
		return
	}

	resp, err := revalidateClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("revalidation: request failed for tag %q: %v", tag, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("revalidation: returned %d for tag %q", resp.StatusCode, tag)
	}
}
