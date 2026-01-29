package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/expr-lang/expr"
	"github.com/essajiwa/hooklab/internal/app/model"
	"github.com/essajiwa/hooklab/internal/app/service"
	"github.com/essajiwa/hooklab/internal/pkg/helpers"
	"github.com/essajiwa/hooklab/internal/pkg/response"
)

// maxBodySize limits request body to 1MB to prevent DoS attacks.
const maxBodySize = 1 << 20 // 1MB

// webhookHandler handles incoming webhook requests at /webhook and /webhook/{key}.
// It stores the event, broadcasts it to SSE subscribers, evaluates rules, and returns
// the appropriate response.
func WebhookHandler(a *service.App, w http.ResponseWriter, r *http.Request) {
	key := helpers.WebhookKeyFromPath(r.URL.Path)
	// Ensure r.Body is not nil for io.ReadAll
	if r.Body == nil {
		r.Body = http.NoBody
	}

	// Read body with size limit
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error reading request body")
		return
	}
	defer r.Body.Close()

	event := a.StoreEvent(r, key, string(body))
	a.BroadcastEvent(event)

	// Try to match a rule first
	ruleConfig, _ := a.EvaluateRules(key, string(body), r.Method, r.Header)
	var config model.ResponseConfig
	if ruleConfig != nil {
		config = *ruleConfig
	} else {
		config = a.GetResponseConfig(key)
	}

	// Create JSON response
	if err := response.SendJSON(w, config.StatusCode, config.Response); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error creating response")
	}
}

// eventsHandler handles GET /api/events requests.
// Returns all stored events, optionally filtered by the "key" query parameter.
func EventsHandler(a *service.App, w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	events := a.GetEvents(key)
	resp := model.EventsResponse{Events: events}
	if err := response.SendJSON(w, http.StatusOK, resp); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error creating response")
	}
}

// responseHandler handles GET and POST requests to /api/response.
// GET returns the current response configuration for a key.
// POST updates the response configuration for a key.
func ResponseHandler(a *service.App, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		key := helpers.ResponseKeyFromRequest(r)
		config := a.GetResponseConfig(key)

		data := map[string]interface{}{
			"response":   config.Response,
			"statusCode": config.StatusCode,
			"key":        key,
		}
		if err := response.SendJSON(w, http.StatusOK, data); err != nil {
			response.SendError(w, http.StatusInternalServerError, "Error creating response")
		}
	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
		if err != nil {
			response.SendError(w, http.StatusInternalServerError, "Error reading request body")
			return
		}
		defer r.Body.Close()

		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			response.SendError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		responseData := payload["response"]
		statusCodeValue, hasStatus := payload["statusCode"]
		key := helpers.ResponseKeyFromRequest(r)
		statusCode := a.GetResponseConfig(key).StatusCode
		if hasStatus {
			if floatVal, ok := statusCodeValue.(float64); ok {
				statusCode = int(floatVal)
			}
		}

		a.SetResponseConfig(key, model.ResponseConfig{
			Response:    responseData,
			ResponseRaw: string(body),
			StatusCode:  statusCode,
		})

		if err := response.SendJSON(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
			response.SendError(w, http.StatusInternalServerError, "Error creating response")
		}
	default:
		response.SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// keysHandler handles GET /api/keys requests.
// Returns a JSON array of all known webhook keys.
func KeysHandler(a *service.App, w http.ResponseWriter, r *http.Request) {
	keys := a.GetKeys()
	data := map[string]interface{}{
		"keys": keys,
	}
	if err := response.SendJSON(w, http.StatusOK, data); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error creating response")
	}
}

// rulesHandler handles CRUD operations for conditional response rules at /api/rules.
// Supports GET (list), POST (create), PUT (update), and DELETE operations.
// The "key" query parameter specifies which webhook key's rules to manage.
func RulesHandler(a *service.App, w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		key = "default"
	}

	switch r.Method {
	case http.MethodGet:
		handleGetRules(a, w, key)
	case http.MethodPost:
		handleCreateRule(a, w, r, key)
	case http.MethodPut:
		handleUpdateRule(a, w, r, key)
	case http.MethodDelete:
		handleDeleteRule(a, w, r, key)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetRules returns all rules for the given webhook key.
func handleGetRules(a *service.App, w http.ResponseWriter, key string) {
	rules := a.GetRules(key)
	data := map[string]interface{}{
		"rules": rules,
		"key":   key,
	}
	if err := response.SendJSON(w, http.StatusOK, data); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error creating response")
	}
}

// handleCreateRule creates a new rule for the given webhook key.
func handleCreateRule(a *service.App, w http.ResponseWriter, r *http.Request, key string) {
	rule, ok := parseAndValidateRule(w, r)
	if !ok {
		return
	}

	created := a.AddRule(key, rule)
	if err := response.SendJSON(w, http.StatusCreated, created); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error creating response")
	}
}

// handleUpdateRule updates an existing rule identified by the "id" query parameter.
func handleUpdateRule(a *service.App, w http.ResponseWriter, r *http.Request, key string) {
	ruleID := r.URL.Query().Get("id")
	if ruleID == "" {
		response.SendError(w, http.StatusBadRequest, "Rule ID required")
		return
	}

	rule, ok := parseAndValidateRule(w, r)
	if !ok {
		return
	}

	if a.UpdateRule(key, ruleID, rule) {
		if err := response.SendJSON(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
			response.SendError(w, http.StatusInternalServerError, "Error creating response")
		}
	} else {
		response.SendError(w, http.StatusNotFound, "Rule not found")
	}
}

// handleDeleteRule removes a rule identified by the "id" query parameter.
func handleDeleteRule(a *service.App, w http.ResponseWriter, r *http.Request, key string) {
	ruleID := r.URL.Query().Get("id")
	if ruleID == "" {
		response.SendError(w, http.StatusBadRequest, "Rule ID required")
		return
	}

	if a.DeleteRule(key, ruleID) {
		if err := response.SendJSON(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
			response.SendError(w, http.StatusInternalServerError, "Error creating response")
		}
	} else {
		response.SendError(w, http.StatusNotFound, "Rule not found")
	}
}

// parseAndValidateRule reads and validates a rule from the request body.
// It validates the expression syntax using the expr library.
// Returns the parsed rule and true on success, or writes an error response and returns false.
func parseAndValidateRule(w http.ResponseWriter, r *http.Request) (model.Rule, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error reading request body")
		return model.Rule{}, false
	}
	defer r.Body.Close()

	var rule model.Rule
	if err := json.Unmarshal(body, &rule); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid JSON")
		return model.Rule{}, false
	}

	if rule.Condition != "" {
		env := map[string]interface{}{
			"body":    map[string]interface{}{},
			"method":  "",
			"headers": map[string][]string{},
		}
		if _, err := expr.Compile(rule.Condition, expr.Env(env), expr.AsBool()); err != nil {
			response.SendError(w, http.StatusBadRequest, "Invalid expression: "+err.Error())
			return model.Rule{}, false
		}
	}

	return rule, true
}