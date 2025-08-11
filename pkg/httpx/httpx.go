package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrEmptyURL         = errors.New("httpx: empty URL")
	ErrInvalidURL       = errors.New("httpx: invalid URL")
	ErrMaxRetries       = errors.New("httpx: max retries reached")
	ErrNonRetryableResp = errors.New("httpx: non-retryable response")
)

type Config struct {
	Timeout        time.Duration
	MaxRetries     int
	BackoffInitial time.Duration
	BackoffMax     time.Duration
	UserAgents     []string
	BaseHeaders    map[string]string
	RetryStatus    []int
	RetryOn        func(status int, err error) bool
}

type Request struct {
	Method  string
	URL     string
	Params  map[string]string
	Headers map[string]string
	Body    io.Reader
}

type Response struct {
	Status  int
	Body    []byte
	Headers http.Header
	URL     string
}

type Client interface {
	Do(ctx context.Context, req Request) (Response, error)
	DoGET(ctx context.Context, rawURL string, params, headers map[string]string) (Response, error)
}

type realClient struct {
	http *http.Client
	cfg  Config
}

func New(cfg Config) Client {
	normalizeConfig(&cfg)

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &realClient{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: tr,
		},
		cfg: cfg,
	}
}

func NewWithHTTP(hc *http.Client, cfg Config) Client {
	normalizeConfig(&cfg)
	if hc == nil {
		return New(cfg)
	}
	return &realClient{http: hc, cfg: cfg}
}

func (c *realClient) DoGET(ctx context.Context, rawURL string, params, headers map[string]string) (Response, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodGet,
		URL:     rawURL,
		Params:  params,
		Headers: headers,
	})
}

func (c *realClient) Do(ctx context.Context, r Request) (Response, error) {
	if r.URL == "" {
		return Response{}, ErrEmptyURL
	}
	if r.Method == "" {
		r.Method = http.MethodGet
	}

	u, err := buildURL(r.URL, r.Params)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, r.Method, u, r.Body)
		if err != nil {
			return Response{}, fmt.Errorf("httpx: build request: %w", err)
		}

		c.setRequestHeaders(req, r.Headers)

		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return Response{}, ctx.Err()
			}
			if c.shouldRetry(0, err) && attempt < c.cfg.MaxRetries {
				c.sleepBackoff(attempt)
				lastErr = err
				continue
			}
			return Response{}, fmt.Errorf("httpx: request failed: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		res := Response{
			Status:  resp.StatusCode,
			Body:    body,
			Headers: resp.Header.Clone(),
			URL:     u,
		}

		if readErr != nil {
			if c.shouldRetry(resp.StatusCode, readErr) && attempt < c.cfg.MaxRetries {
				c.sleepBackoff(attempt)
				lastErr = readErr
				continue
			}
			return res, fmt.Errorf("httpx: read body: %w", readErr)
		}

		if c.shouldRetry(resp.StatusCode, nil) && attempt < c.cfg.MaxRetries {
			lastErr = fmt.Errorf("httpx: retryable status %d", resp.StatusCode)
			c.sleepBackoff(attempt)
			continue
		}

		if c.shouldRetry(resp.StatusCode, nil) && attempt > 0 && attempt >= c.cfg.MaxRetries {
			return Response{}, fmt.Errorf("%w: retryable status %d", ErrMaxRetries, resp.StatusCode)
		}

		return res, nil
	}

	return Response{}, fmt.Errorf("%w: %v", ErrMaxRetries, lastErr)
}

func (c *realClient) setRequestHeaders(req *http.Request, customHeaders map[string]string) {
	for k, v := range c.cfg.BaseHeaders {
		req.Header.Set(k, v)
	}

	if _, ok := headerLookup(customHeaders, "User-Agent"); !ok {
		req.Header.Set("User-Agent", c.pickUA())
	}

	if _, ok := headerLookup(customHeaders, "Accept"); !ok {
		req.Header.Set("Accept", "*/*")
	}

	if _, ok := headerLookup(customHeaders, "Accept-Language"); !ok {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	}

	for k, v := range customHeaders {
		req.Header.Set(k, v)
	}
}

func (c *realClient) shouldRetry(status int, err error) bool {
	if c.cfg.RetryOn != nil {
		return c.cfg.RetryOn(status, err)
	}
	if err != nil {
		return true
	}
	for _, s := range c.cfg.RetryStatus {
		if status == s {
			return true
		}
	}
	return false
}

func (c *realClient) sleepBackoff(attempt int) {
	backoff := float64(c.cfg.BackoffInitial) * math.Pow(2, float64(attempt))
	backoff += float64(time.Duration(rand.Intn(250)) * time.Millisecond)
	delay := time.Duration(backoff)
	if delay > c.cfg.BackoffMax {
		delay = c.cfg.BackoffMax
	}
	time.Sleep(delay)
}

func (c *realClient) pickUA() string {
	if len(c.cfg.UserAgents) == 0 {
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
	}
	return c.cfg.UserAgents[rand.Intn(len(c.cfg.UserAgents))]
}

func normalizeConfig(cfg *Config) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.BackoffInitial <= 0 {
		cfg.BackoffInitial = time.Second
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = 30 * time.Second
	}
	if len(cfg.RetryStatus) == 0 && cfg.RetryOn == nil {
		cfg.RetryStatus = []int{http.StatusTooManyRequests}
		for code := 500; code <= 599; code++ {
			cfg.RetryStatus = append(cfg.RetryStatus, code)
		}
	}
}

func buildURL(raw string, params map[string]string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func headerLookup(h map[string]string, key string) (string, bool) {
	for k, v := range h {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
