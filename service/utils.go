package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

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

func toInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
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

func toBool(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func fetchJSON(ctx context.Context, url string) (map[string]any, error) {
	if url == "" {
		return nil, errors.New("url is empty")
	}
	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func dataObject(payload map[string]any) (map[string]any, error) {
	data, ok := payload["data"].(map[string]any)
	if !ok {
		return nil, errors.New("payload missing data object")
	}
	return data, nil
}

func requiredInt(m map[string]any, key string) (int, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("missing %s", key)
	}
	return toInt(v), nil
}

// currentEpochFromPayload reads Dora's /api/v1/epochs body.
// The epoch is data.current_epoch. The top-level object has no epoch field.
func currentEpochFromPayload(payload map[string]any) (int, error) {
	data, err := dataObject(payload)
	if err != nil {
		return 0, err
	}
	return requiredInt(data, "current_epoch")
}
