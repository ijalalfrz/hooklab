package helpers

import (
	"net/http"
	"strings"
)

// webhookKeyFromPath extracts the webhook key from a URL path.
// Returns "default" if no key is specified.
func WebhookKeyFromPath(path string) string {
	key := strings.TrimPrefix(path, "/webhook")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "default"
	}
	return key
}

// responseKeyFromRequest extracts the response key from a request.
// Checks the "key" query parameter first, then the URL path.
func ResponseKeyFromRequest(r *http.Request) string {
	if key := r.URL.Query().Get("key"); key != "" {
		return key
	}
	key := strings.TrimPrefix(r.URL.Path, "/api/response")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "default"
	}
	return key
}