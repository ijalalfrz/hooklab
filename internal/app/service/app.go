package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/essajiwa/hooklab/internal/app/model"
)

// App holds the application state including webhook events, response configurations,
// conditional rules, and SSE subscribers. All fields are protected by a mutex for
// concurrent access safety.
type App struct {
	Responses   map[string]model.ResponseConfig
	Rules       map[string][]model.Rule // rules per webhook key
	Mu          sync.Mutex
	Events      []model.Event
	LastID      int
	RuleLastID  int
	Subscribers map[chan model.Event]struct{}
}

// StoreEvent captures an incoming webhook request and stores it in memory.
// It maintains a maximum of 50 events, discarding the oldest when the limit is reached.
func (a *App) StoreEvent(r *http.Request, key, body string) model.Event {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	a.LastID++
	event := model.Event{
		ID:        a.LastID,
		Timestamp: time.Now(),
		Method:    r.Method,
		Path:      r.URL.Path,
		Key:       key,
		Headers:   r.Header,
		Body:      body,
	}

	const maxEvents = 50
	a.Events = append([]model.Event{event}, a.Events...)
	if len(a.Events) > maxEvents {
		a.Events = a.Events[:maxEvents]
	}

	return event
}

// GetResponseConfig returns the response configuration for the given webhook key.
// If no configuration exists for the key, it falls back to "default", then to a
// hardcoded fallback response.
func (a *App) GetResponseConfig(key string) model.ResponseConfig {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if a.Responses == nil {
		a.Responses = make(map[string]model.ResponseConfig)
	}

	if config, ok := a.Responses[key]; ok {
		return config
	}

	// Return default config if key not found
	if defaultConfig, ok := a.Responses["default"]; ok {
		return defaultConfig
	}

	// Fallback if no default exists
	return model.ResponseConfig{
		Response:   map[string]string{"result": "ok"},
		StatusCode: 200,
	}
}

// SetResponseConfig stores a response configuration for the given webhook key.
// An empty key defaults to "default".
func (a *App) SetResponseConfig(key string, config model.ResponseConfig) {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if a.Responses == nil {
		a.Responses = make(map[string]model.ResponseConfig)
	}
	if key == "" {
		key = "default"
	}
	a.Responses[key] = config
}

// AddSubscriber creates a new SSE subscriber channel and registers it.
// Events will be broadcast to this channel until RemoveSubscriber is called.
func (a *App) AddSubscriber() chan model.Event {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if a.Subscribers == nil {
		a.Subscribers = make(map[chan model.Event]struct{})
	}

	ch := make(chan model.Event, 1)
	a.Subscribers[ch] = struct{}{}
	return ch
}

// RemoveSubscriber unregisters an SSE subscriber and closes its channel.
func (a *App) RemoveSubscriber(ch chan model.Event) {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if _, ok := a.Subscribers[ch]; !ok {
		return
	}
	delete(a.Subscribers, ch)
	close(ch)
}

// BroadcastEvent sends an event to all registered SSE subscribers.
// Non-blocking: if a subscriber's channel is full, the event is dropped for that subscriber.
func (a *App) BroadcastEvent(event model.Event) {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	for ch := range a.Subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

// CloseSubscribers closes all SSE subscriber channels during shutdown.
func (a *App) CloseSubscribers() {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	for ch := range a.Subscribers {
		close(ch)
	}
	a.Subscribers = make(map[chan model.Event]struct{})
}

// GetKeys returns a sorted list of all known webhook keys.
// Keys are collected from events, responses, and rules. The "default" key is always included.
func (a *App) GetKeys() []string {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	keySet := make(map[string]struct{})

	// Add keys from events
	for _, event := range a.Events {
		keySet[event.Key] = struct{}{}
	}

	// Add keys from responses
	for key := range a.Responses {
		keySet[key] = struct{}{}
	}

	// Add keys from rules
	for key := range a.Rules {
		keySet[key] = struct{}{}
	}

	// Always include "default"
	keySet["default"] = struct{}{}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

// GetRules returns all rules for the given webhook key, sorted by priority (ascending).
// Lower priority values are evaluated first.
func (a *App) GetRules(key string) []model.Rule {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if a.Rules == nil {
		return []model.Rule{}
	}

	rules := a.Rules[key]
	if rules == nil {
		return []model.Rule{}
	}

	// Return sorted by priority
	sorted := make([]model.Rule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}

// SetRules replaces all rules for the given webhook key.
func (a *App) SetRules(key string, rules []model.Rule) {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if a.Rules == nil {
		a.Rules = make(map[string][]model.Rule)
	}
	a.Rules[key] = rules
}

// AddRule adds a new rule for the given webhook key and assigns it a unique ID.
func (a *App) AddRule(key string, rule model.Rule) model.Rule {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if a.Rules == nil {
		a.Rules = make(map[string][]model.Rule)
	}

	a.RuleLastID++
	rule.ID = fmt.Sprintf("rule_%d", a.RuleLastID)

	a.Rules[key] = append(a.Rules[key], rule)
	return rule
}

// UpdateRule updates an existing rule by ID. Returns true if the rule was found and updated.
func (a *App) UpdateRule(key string, ruleID string, updated model.Rule) bool {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if a.Rules == nil {
		return false
	}

	rules := a.Rules[key]
	for i, r := range rules {
		if r.ID == ruleID {
			updated.ID = ruleID
			rules[i] = updated
			a.Rules[key] = rules
			return true
		}
	}
	return false
}

// DeleteRule removes a rule by ID. Returns true if the rule was found and deleted.
func (a *App) DeleteRule(key string, ruleID string) bool {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if a.Rules == nil {
		return false
	}

	rules := a.Rules[key]
	for i, r := range rules {
		if r.ID == ruleID {
			a.Rules[key] = append(rules[:i], rules[i+1:]...)
			return true
		}
	}
	return false
}

// GetEvents returns all events or filtered by key.
func (a *App) GetEvents(key string) []model.Event {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	if key == "" {
		return append([]model.Event(nil), a.Events...)
	}

	filtered := make([]model.Event, 0, len(a.Events))
	for _, event := range a.Events {
		if event.Key == key {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// EvaluateRules checks all enabled rules for a key and returns the first matching response.
// Rules are evaluated in priority order. The expression environment includes:
//   - body: parsed JSON body (or raw string if not valid JSON)
//   - method: HTTP method string
//   - headers: map of header names to values
//
// Returns nil if no rule matches.
func (a *App) EvaluateRules(key string, body string, method string, headers map[string][]string) (*model.ResponseConfig, error) {
	rules := a.GetRules(key)

	// Parse body as JSON for expression evaluation
	var bodyData interface{}
	if body != "" {
		if err := json.Unmarshal([]byte(body), &bodyData); err != nil {
			// If body is not valid JSON, use it as a string
			bodyData = body
		}
	}

	// Build environment for expression evaluation
	env := map[string]interface{}{
		"body":    bodyData,
		"method":  method,
		"headers": headers,
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Compile and evaluate the expression
		program, err := expr.Compile(rule.Condition, expr.Env(env), expr.AsBool())
		if err != nil {
			continue // Skip invalid expressions
		}

		result, err := expr.Run(program, env)
		if err != nil {
			continue
		}

		if matched, ok := result.(bool); ok && matched {
			return &model.ResponseConfig{
				Response:   rule.Response,
				StatusCode: rule.StatusCode,
			}, nil
		}
	}

	return nil, nil // No rule matched
}