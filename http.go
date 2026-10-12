package pidgely

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	backoffBase   = 500 * time.Millisecond
	backoffCap    = 8 * time.Second
	maxRetryAfter = 60 * time.Second
)

// RequestOption configures a single request.
type RequestOption func(*requestConfig)

type requestConfig struct {
	idempotencyKey string
	headers        http.Header
	maxRetries     *int
	timeout        *time.Duration
}

// WithIdempotencyKey sends the Idempotency-Key header. For 24 hours the API
// replays the stored response for the same key (with the header
// idempotent-replayed: true) and answers 409 when the body differs. It also makes
// a POST request safe to retry automatically.
func WithIdempotencyKey(key string) RequestOption {
	return func(rc *requestConfig) { rc.idempotencyKey = key }
}

// WithRequestHeader adds a header to this request.
func WithRequestHeader(key, value string) RequestOption {
	return func(rc *requestConfig) { rc.headers.Add(key, value) }
}

// WithRequestRetries overrides the client's maximum number of retries for this request.
func WithRequestRetries(n int) RequestOption {
	return func(rc *requestConfig) {
		if n >= 0 {
			rc.maxRetries = &n
		}
	}
}

// WithRequestTimeout overrides the client's per-attempt timeout for this request.
func WithRequestTimeout(d time.Duration) RequestOption {
	return func(rc *requestConfig) {
		if d >= 0 {
			rc.timeout = &d
		}
	}
}

var retryableStatus = map[int]bool{408: true, 429: true, 500: true, 502: true, 503: true, 504: true}

type attemptResult struct {
	status  int
	headers http.Header
	body    []byte
	err     *Error // set for API and network errors
	network bool   // no response was received
}

// do performs a request with retries and decodes a successful JSON body into out.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any, opts []RequestOption) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.apiKey == "" {
		return &Error{Name: ErrMissingAPIKey, Message: "missing API key: pass it to pidgely.NewClient or set PIDGELY_API_KEY"}
	}
	rc := requestConfig{headers: http.Header{}}
	for _, opt := range opts {
		opt(&rc)
	}

	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return &Error{Name: ErrApplication, Message: "unable to encode the request body: " + err.Error(), Err: err}
		}
	}

	maxRetries := c.maxRetries
	if rc.maxRetries != nil {
		maxRetries = *rc.maxRetries
	}
	timeout := c.timeout
	if rc.timeout != nil {
		timeout = *rc.timeout
	}
	// A POST is only retried with an idempotency key: the API might already
	// have accepted it, and retrying without a key could send an email twice.
	canRetry := method != http.MethodPost || rc.idempotencyKey != ""

	for attempt := 0; ; attempt++ {
		res := c.attempt(ctx, method, u, payload, &rc, timeout)
		if res.err == nil {
			if out != nil && len(bytes.TrimSpace(res.body)) > 0 {
				if err := json.Unmarshal(res.body, out); err != nil {
					return &Error{Name: ErrApplication, Message: "unable to parse the API response", Headers: res.headers, Err: err}
				}
			}
			return nil
		}
		if attempt >= maxRetries || ctx.Err() != nil {
			return res.err
		}
		delay, ok := retryDelay(res, attempt, canRetry)
		if !ok {
			return res.err
		}
		if err := c.sleep(ctx, delay); err != nil {
			return &Error{Name: ErrApplication, Message: "request canceled: " + err.Error(), Err: err}
		}
	}
}

func retryDelay(res attemptResult, attempt int, canRetry bool) (time.Duration, bool) {
	if !canRetry {
		return 0, false
	}
	if res.network {
		return backoff(attempt), true
	}
	if !retryableStatus[res.status] || strings.HasSuffix(res.err.Name, "quota_exceeded") {
		return 0, false
	}
	if d, ok := parseRetryAfter(res.headers.Get("Retry-After"), time.Now()); ok {
		if d > maxRetryAfter {
			return 0, false
		}
		return d, true
	}
	return backoff(attempt), true
}

func (c *Client) attempt(ctx context.Context, method, u string, payload []byte, rc *requestConfig, timeout time.Duration) attemptResult {
	actx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		actx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(actx, method, u, reader)
	if err != nil {
		return attemptResult{network: true, err: &Error{Name: ErrApplication, Message: "invalid request: " + sanitize(err.Error(), c.apiKey), Err: err}}
	}
	for k, v := range c.headers {
		req.Header[k] = append([]string(nil), v...)
	}
	for k, v := range rc.headers {
		req.Header[k] = append([]string(nil), v...)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	ua := "pidgely-go/" + Version
	if c.userAgent != "" {
		ua += " " + c.userAgent
	}
	req.Header.Set("User-Agent", ua)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if rc.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", rc.idempotencyKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return attemptResult{network: true, err: networkError(ctx, actx, err, timeout, c.apiKey)}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return attemptResult{network: true, err: networkError(ctx, actx, err, timeout, c.apiKey)}
	}
	res := attemptResult{status: resp.StatusCode, headers: resp.Header, body: data}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		res.err = parseError(resp.StatusCode, resp.Header, data)
	}
	return res
}

func networkError(parent, attemptCtx context.Context, err error, timeout time.Duration, apiKey string) *Error {
	switch {
	case parent.Err() != nil:
		return &Error{Name: ErrApplication, Message: "request canceled: " + parent.Err().Error(), Err: parent.Err()}
	case errors.Is(attemptCtx.Err(), context.DeadlineExceeded):
		return &Error{Name: ErrApplication, Message: fmt.Sprintf("request timed out after %s", timeout), Err: context.DeadlineExceeded}
	default:
		return &Error{Name: ErrApplication, Message: "unable to reach the Pidgely API: " + sanitize(err.Error(), apiKey), Err: err}
	}
}

// sanitize removes the API key from a message, as a defence in depth.
func sanitize(msg, apiKey string) string {
	if apiKey == "" {
		return msg
	}
	return strings.ReplaceAll(msg, apiKey, "[REDACTED]")
}

func parseError(status int, headers http.Header, data []byte) *Error {
	e := &Error{StatusCode: status, Headers: headers}
	var body struct {
		StatusCode *int            `json:"statusCode"`
		Name       *string         `json:"name"`
		Message    *string         `json:"message"`
		Error      json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &body) == nil {
		if body.Name != nil && body.Message != nil {
			e.Name, e.Message = *body.Name, *body.Message
			return e
		}
		// Native /v1 shape: {"error": {"code", "message"}}
		var native struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if len(body.Error) > 0 && json.Unmarshal(body.Error, &native) == nil && native.Code != "" {
			e.Name, e.Message = native.Code, native.Code
			if native.Message != "" {
				e.Message = native.Message
			}
			return e
		}
	}
	if status >= 500 {
		e.Name = ErrInternalServer
	} else {
		e.Name = ErrApplication
	}
	e.Message = strings.TrimSpace(string(data))
	if e.Message == "" {
		e.Message = fmt.Sprintf("HTTP %d", status)
	}
	return e
}

// parseRetryAfter reads Retry-After in seconds or as an HTTP date.
func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.ParseFloat(v, 64); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs * float64(time.Second)), true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// backoff is exponential with jitter: 50-100% of min(cap, base*2^attempt).
func backoff(attempt int) time.Duration {
	exp := backoffCap
	if attempt < 5 {
		exp = min(backoffCap, backoffBase<<attempt)
	}
	half := exp / 2
	return half + time.Duration(rand.Int64N(int64(half)+1))
}

// seg escapes a path segment (ids, emails, template aliases).
func seg(s string) string { return url.PathEscape(s) }

// ListParams are the cursor pagination parameters shared by list endpoints.
// After and Before take an item id; set at most one of them.
type ListParams struct {
	// Limit is the page size (1-100).
	Limit int
	// After returns items after this id.
	After string
	// Before returns items before this id.
	Before string
}

func (p *ListParams) values() url.Values {
	q := url.Values{}
	if p == nil {
		return q
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.After != "" {
		q.Set("after", p.After)
	}
	if p.Before != "" {
		q.Set("before", p.Before)
	}
	return q
}

// List is a page of results.
type List[T any] struct {
	Object  string `json:"object"`
	HasMore bool   `json:"has_more"`
	Data    []T    `json:"data"`
}

// ObjectRef is returned by most mutations.
type ObjectRef struct {
	Object string `json:"object"`
	Id     string `json:"id"`
}

// IDResponse is a bare {"id": ...} response.
type IDResponse struct {
	Id string `json:"id"`
}

// Deleted is returned by deletions.
type Deleted struct {
	Object  string `json:"object"`
	Id      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// Data wraps a list of results without pagination.
type Data[T any] struct {
	Data []T `json:"data"`
}

func get[T any](ctx context.Context, c *Client, path string, query url.Values, opts []RequestOption) (*T, error) {
	out := new(T)
	if err := c.do(ctx, http.MethodGet, path, query, nil, out, opts); err != nil {
		return nil, err
	}
	return out, nil
}

func send[T any](ctx context.Context, c *Client, method, path string, body any, opts []RequestOption) (*T, error) {
	out := new(T)
	if err := c.do(ctx, method, path, nil, body, out, opts); err != nil {
		return nil, err
	}
	return out, nil
}

func required(name, value string) error {
	if value == "" {
		return &Error{Name: ErrMissingRequiredField, Message: "missing `" + name + "`"}
	}
	return nil
}
