package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cfg := Config{
		Timeout:        5 * time.Second,
		MaxRetries:     2,
		BackoffInitial: 100 * time.Millisecond,
		BackoffMax:     1 * time.Second,
	}

	client := New(cfg)
	if client == nil {
		t.Fatal("expected client to be created")
	}

	realClient, ok := client.(*realClient)
	if !ok {
		t.Fatal("expected realClient type")
	}

	if realClient.cfg.Timeout != cfg.Timeout {
		t.Errorf("expected timeout %v, got %v", cfg.Timeout, realClient.cfg.Timeout)
	}
}

func TestNewWithHTTP(t *testing.T) {
	cfg := Config{Timeout: 5 * time.Second}
	httpClient := &http.Client{Timeout: 10 * time.Second}

	client := NewWithHTTP(httpClient, cfg)
	if client == nil {
		t.Fatal("expected client to be created")
	}

	realClient, ok := client.(*realClient)
	if !ok {
		t.Fatal("expected realClient type")
	}

	if realClient.http != httpClient {
		t.Error("expected custom http client to be used")
	}
}

func TestNewWithHTTPNil(t *testing.T) {
	cfg := Config{Timeout: 5 * time.Second}
	client := NewWithHTTP(nil, cfg)
	if client == nil {
		t.Fatal("expected client to be created")
	}
}

func TestNormalizeConfig(t *testing.T) {
	cfg := Config{}
	normalizeConfig(&cfg)

	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", cfg.Timeout)
	}
	if cfg.MaxRetries != 0 {
		t.Errorf("expected default max retries 0, got %d", cfg.MaxRetries)
	}
	if cfg.BackoffInitial != time.Second {
		t.Errorf("expected default backoff initial 1s, got %v", cfg.BackoffInitial)
	}
	if cfg.BackoffMax != 30*time.Second {
		t.Errorf("expected default backoff max 30s, got %v", cfg.BackoffMax)
	}
	if len(cfg.RetryStatus) == 0 {
		t.Error("expected retry status to be populated")
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		params map[string]string
		want   string
	}{
		{
			name:   "simple url",
			raw:    "https://example.com",
			params: nil,
			want:   "https://example.com",
		},
		{
			name:   "url with params",
			raw:    "https://example.com",
			params: map[string]string{"key": "value", "foo": "bar"},
			want:   "https://example.com?foo=bar&key=value",
		},
		{
			name:   "url with existing query",
			raw:    "https://example.com?existing=1",
			params: map[string]string{"key": "value"},
			want:   "https://example.com?existing=1&key=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildURL(tt.raw, tt.params)
			if err != nil {
				t.Fatalf("buildURL() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("buildURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildURLInvalid(t *testing.T) {
	_, err := buildURL("://invalid", nil)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestHeaderLookup(t *testing.T) {
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "*/*",
	}

	tests := []struct {
		name     string
		key      string
		expected string
		found    bool
	}{
		{"exact match", "Content-Type", "application/json", true},
		{"case insensitive", "content-type", "application/json", true},
		{"not found", "X-Not-Found", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := headerLookup(headers, tt.key)
			if got != tt.expected {
				t.Errorf("headerLookup() = %v, want %v", got, tt.expected)
			}
			if found != tt.found {
				t.Errorf("headerLookup() found = %v, want %v", found, tt.found)
			}
		})
	}
}

func TestPickUA(t *testing.T) {
	client := &realClient{
		cfg: Config{
			UserAgents: []string{"UA1", "UA2", "UA3"},
		},
	}

	ua := client.pickUA()
	if ua == "" {
		t.Error("expected user agent to be returned")
	}

	client.cfg.UserAgents = nil
	ua = client.pickUA()
	if ua == "" {
		t.Error("expected default user agent to be returned")
	}
}

func TestDoGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer server.Close()

	client := New(Config{Timeout: 5 * time.Second})
	resp, err := client.DoGET(context.Background(), server.URL, nil, nil)
	if err != nil {
		t.Fatalf("DoGET() error = %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
	if string(resp.Body) != "response" {
		t.Errorf("expected body 'response', got '%s'", string(resp.Body))
	}
}

func TestDoEmptyURL(t *testing.T) {
	client := New(Config{})
	_, err := client.Do(context.Background(), Request{})
	if !errors.Is(err, ErrEmptyURL) {
		t.Errorf("expected ErrEmptyURL, got %v", err)
	}
}

func TestDoInvalidURL(t *testing.T) {
	client := New(Config{})
	_, err := client.Do(context.Background(), Request{URL: "://invalid"})
	if !errors.Is(err, ErrInvalidURL) {
		t.Errorf("expected ErrInvalidURL, got %v", err)
	}
}

func TestDoWithRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := New(Config{
		Timeout:        5 * time.Second,
		MaxRetries:     3,
		BackoffInitial: 10 * time.Millisecond,
		BackoffMax:     100 * time.Millisecond,
	})

	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		URL:    server.URL,
	})

	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
	if attempts < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempts)
	}
}

func TestDoNoRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(Config{
		Timeout:        5 * time.Second,
		MaxRetries:     0,
		BackoffInitial: 10 * time.Millisecond,
		BackoffMax:     100 * time.Millisecond,
	})

	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		URL:    server.URL,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.Status)
	}
	if attempts != 1 {
		t.Errorf("expected exactly 1 attempt, got %d", attempts)
	}
}

func TestDoMaxRetriesExceeded(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// Always return 500 (retryable) to ensure we hit max retries
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(Config{
		Timeout:        5 * time.Second,
		MaxRetries:     1,
		BackoffInitial: 10 * time.Millisecond,
		BackoffMax:     100 * time.Millisecond,
	})

	_, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		URL:    server.URL,
	})

	if !errors.Is(err, ErrMaxRetries) {
		t.Errorf("expected ErrMaxRetries, got %v", err)
	}
	if attempts < 2 {
		t.Errorf("expected at least 2 attempts, got %d", attempts)
	}
}

func TestDoWithCustomRetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := New(Config{
		Timeout:        5 * time.Second,
		MaxRetries:     3,
		BackoffInitial: 10 * time.Millisecond,
		BackoffMax:     100 * time.Millisecond,
		RetryOn: func(status int, err error) bool {
			return status == http.StatusBadRequest
		},
	})

	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		URL:    server.URL,
	})

	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
}

func TestDoWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "value" {
			t.Error("expected X-Custom header to be set")
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("expected User-Agent header to be set")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(Config{
		Timeout: 5 * time.Second,
		BaseHeaders: map[string]string{
			"X-Base": "base-value",
		},
	})

	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		URL:    server.URL,
		Headers: map[string]string{
			"X-Custom": "value",
		},
	})

	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
}

func TestDoWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "test body" {
			t.Errorf("expected body 'test body', got '%s'", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(Config{Timeout: 5 * time.Second})
	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodPost,
		URL:    server.URL,
		Body:   strings.NewReader("test body"),
	})

	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
}

func TestDoContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := New(Config{Timeout: 5 * time.Second})
	_, err := client.Do(ctx, Request{
		Method: http.MethodGet,
		URL:    server.URL,
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestShouldRetry(t *testing.T) {
	client := &realClient{
		cfg: Config{
			RetryStatus: []int{500, 502, 503},
		},
	}

	tests := []struct {
		name     string
		status   int
		err      error
		expected bool
	}{
		{"retryable status", 500, nil, true},
		{"non-retryable status", 200, nil, false},
		{"network error", 0, errors.New("network error"), true},
		{"nil error", 200, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.shouldRetry(tt.status, tt.err)
			if got != tt.expected {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShouldRetryWithCustomLogic(t *testing.T) {
	client := &realClient{
		cfg: Config{
			RetryOn: func(status int, err error) bool {
				return status == 418
			},
		},
	}

	if !client.shouldRetry(418, nil) {
		t.Error("expected retry for status 418")
	}
	if client.shouldRetry(200, nil) {
		t.Error("expected no retry for status 200")
	}
}

func TestSleepBackoff(t *testing.T) {
	client := &realClient{
		cfg: Config{
			BackoffInitial: 10 * time.Millisecond,
			BackoffMax:     100 * time.Millisecond,
		},
	}

	start := time.Now()
	client.sleepBackoff(2)
	duration := time.Since(start)

	if duration < 10*time.Millisecond {
		t.Error("expected backoff to take at least initial duration")
	}
	if duration > 200*time.Millisecond {
		t.Error("expected backoff to not exceed reasonable time")
	}
}
