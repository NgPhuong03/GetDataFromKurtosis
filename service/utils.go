package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"context"
	"strconv"
)

// toString converts a JSON value to string safely.
// Params: v is any JSON-decoded value (string, number, etc.).
// Returns: string representation, or empty string if cannot convert.
// Usage: s := toString(val["name"]).
func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	default:
		return ""
	}
	}
	
	// toInt converts a JSON value to int safely.
	// Params: v is any JSON-decoded value (json.Number, float64, string numeric).
	// Returns: int value, or 0 if cannot convert.
	// Usage: n := toInt(val["index"]).
	func toInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	case json.Number:
		if i64, err := t.Int64(); err == nil {
			return int(i64)
		}
		if f64, err := t.Float64(); err == nil {
			return int(f64)
		}
		return 0
	case string:
		if i64, err := strconv.ParseInt(t, 10, 64); err == nil {
			return int(i64)
		}
		if f64, err := strconv.ParseFloat(t, 64); err == nil {
			return int(f64)
		}
		return 0
	default:
		return 0
	}
	}

// toBool converts a JSON value to bool safely.
// Params: v is any JSON-decoded value (bool, string).
// Returns: bool value, or false if cannot convert.
// Usage: b := toBool(val["finalized"]).
func toBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	default:
		return false
	}
}
	
	
	

// fetchJSON performs an HTTP GET to the provided URL and decodes the response as JSON.
// Params: ctx for timeout, url is the target endpoint.
// Returns: a generic map[string]any representing the JSON payload.
// Usage: payload, err := fetchJSON(ctx, targetURL)
func fetchJSON(ctx context.Context, url string) (map[string]any, error) {
	if url == "" {
			return nil, errors.New("TARGET_URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
			return nil, err
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
			return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return nil, errors.New("non-2xx status code from target API")
	}

	dec := json.NewDecoder(resp.Body)
	dec.UseNumber() // preserve numbers as json.Number to avoid float64 rounding
	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}
