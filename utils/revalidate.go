package utils

import (
	"bytes"
	"encoding/json"
	"io"
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

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("revalidation: failed to create request for tag %q: %v", tag, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Bypass Vercel Deployment Protection on staging/preview deployments.
	if bypassSecret := os.Getenv("VERCEL_AUTOMATION_BYPASS_SECRET"); bypassSecret != "" {
		req.Header.Set("x-vercel-protection-bypass", bypassSecret)
	}

	resp, err := revalidateClient.Do(req)
	if err != nil {
		log.Printf("revalidation: request failed for tag %q: %v", tag, err)
		return
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("revalidation: returned %d for tag %q", resp.StatusCode, tag)
	}
}
