package model

import (
	"time"
)

// ResponseConfig defines the response to return for a webhook request.
// Response can be any JSON-serializable value, and StatusCode is the HTTP status.
type ResponseConfig struct {
	Response    interface{} // JSON response body
	ResponseRaw string      // Raw JSON string of the response
	StatusCode  int         // HTTP status code (e.g., 200, 404)
}

// Rule represents a conditional response rule that can override the default response
// based on request content. Rules are evaluated using the expr expression language.
type Rule struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Condition  string      `json:"condition"` // expr expression, e.g., "body.amount > 100"
	Response   interface{} `json:"response"`
	StatusCode int         `json:"statusCode"`
	Priority   int         `json:"priority"` // Lower = higher priority
	Enabled    bool        `json:"enabled"`
}

// Event represents a captured webhook request with all its metadata.
// Events are stored in memory and broadcast to SSE subscribers in real-time.
type Event struct {
	ID        int                 `json:"id"`        // Unique event identifier
	Timestamp time.Time           `json:"timestamp"` // When the event was received
	Method    string              `json:"method"`    // HTTP method (GET, POST, etc.)
	Path      string              `json:"path"`      // Request path
	Key       string              `json:"key"`       // Webhook key from path
	Headers   map[string][]string `json:"headers"`   // Request headers
	Body      string              `json:"body"`      // Request body
}

// EventsResponse is the JSON response structure for the /api/events endpoint.
type EventsResponse struct {
	Events []Event `json:"events"`
}