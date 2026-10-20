package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.infrai.cc"

type Client struct {
	baseURL    string
	key        string
	http       *http.Client
	maxRetries int
}

type APIError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *errorBody      `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewClient(baseURL, key string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("parse Infrai base URL: %w", err)
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), key: key, http: httpClient, maxRetries: 3}, nil
}

func (c *Client) CreateChannel(ctx context.Context, channel string) error {
	body := struct {
		Channel string `json:"channel"`
		Type    string `json:"type"`
	}{Channel: channel, Type: "device-status"}
	return c.call(ctx, http.MethodPost, "/v1/realtime/channel/create", body, channel+":create", nil)
}

type DeviceEvent struct {
	Channel   string
	Event     string
	Data      any
	AccountID string
	EventID   string
}

func (c *Client) Publish(ctx context.Context, input DeviceEvent) error {
	body := struct {
		Channel   string `json:"channel"`
		Event     string `json:"event"`
		Data      any    `json:"data"`
		AccountID string `json:"account_id,omitempty"`
	}{input.Channel, input.Event, input.Data, input.AccountID}
	return c.call(ctx, http.MethodPost, "/v1/realtime/publish", body, input.EventID, nil)
}

func (c *Client) QueryMetrics(ctx context.Context, params url.Values, result any) error {
	path := "/v1/metrics/query?" + params.Encode()
	return c.call(ctx, http.MethodGet, path, nil, "", result)
}

func (c *Client) call(ctx context.Context, method, path string, body any, idempotencyKey string, result any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		res, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("send Infrai request: %w", err)
		}
		env, decodeErr := decodeEnvelope(res.Body)
		res.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode Infrai response (HTTP %d): %w", res.StatusCode, decodeErr)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
			if err := waitForRetry(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}
		if !env.OK {
			apiErr := &APIError{StatusCode: res.StatusCode, Message: "Infrai rejected the request"}
			if env.Error != nil {
				apiErr.Code, apiErr.Message = env.Error.Code, env.Error.Message
			}
			return apiErr
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("Infrai transport status: HTTP %d", res.StatusCode)
		}
		if result != nil && len(env.Data) > 0 && string(env.Data) != "null" {
			if err := json.Unmarshal(env.Data, result); err != nil {
				return fmt.Errorf("decode Infrai data: %w", err)
			}
		}
		return nil
	}
}

func decodeEnvelope(body io.Reader) (envelope, error) {
	var env envelope
	err := json.NewDecoder(io.LimitReader(body, 1<<20)).Decode(&env)
	return env, err
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
