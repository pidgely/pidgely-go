// Package pidgely is the official Go SDK for the Pidgely email API
// (https://pidgely.com), which is compatible with the Resend REST API.
//
//	client := pidgely.NewClient(os.Getenv("PIDGELY_API_KEY"))
//
//	sent, err := client.Emails.Send(ctx, &pidgely.SendEmailRequest{
//		From:    "Acme <hello@acme.com>",
//		To:      []string{"ana@example.org"},
//		Subject: "Welcome to Acme",
//		Html:    "<p>Your account is ready.</p>",
//	})
//
// Every method takes a context, returns typed structs, and reports API, network
// and timeout failures as a *[Error]. Full documentation lives at
// https://docs.pidgely.com.
package pidgely

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// Version is the SDK version, sent in the User-Agent header as pidgely-go/<Version>.
const Version = "0.1.0"

// DefaultBaseURL is the API base URL used when neither [WithBaseURL] nor
// PIDGELY_BASE_URL is set.
const DefaultBaseURL = "https://api.pidgely.com"

const (
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 2
)

// Client is a Pidgely API client. It is safe for concurrent use.
//
// The API key is never included in the client's string, Go-syntax or slog
// representation, nor in any error the client returns.
type Client struct {
	// Emails sends and manages individual emails.
	Emails *EmailsService
	// Batch sends up to 100 emails in one request.
	Batch *BatchService
	// Domains manages sending domains and their DNS records.
	Domains *DomainsService
	// ApiKeys manages API keys.
	ApiKeys *ApiKeysService
	// Contacts manages contacts and their segment memberships.
	Contacts *ContactsService
	// Segments manages contact segments.
	Segments *SegmentsService
	// Audiences is the legacy name of Segments, kept for Resend compatibility.
	//
	// Deprecated: use Segments.
	Audiences *SegmentsService
	// Broadcasts creates and sends marketing broadcasts.
	Broadcasts *BroadcastsService
	// Templates manages reusable email templates.
	Templates *TemplatesService
	// Webhooks manages webhook endpoints and verifies deliveries.
	Webhooks *WebhooksService
	// Suppressions manages the suppression list.
	Suppressions *SuppressionsService

	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
	userAgent  string
	headers    http.Header
	sleep      func(ctx context.Context, d time.Duration) error
}

// Option configures a [Client].
type Option func(*Client)

// WithBaseURL sets the API base URL (default PIDGELY_BASE_URL or https://api.pidgely.com).
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient sets the *http.Client used for requests (default: a new client
// with no global timeout; per-attempt timeouts are set with [WithTimeout]).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithRetries sets the maximum number of automatic retries (default 2, 0 disables).
//
// 408, 429, 500, 502, 503 and 504 responses, network errors and timeouts are
// retried with exponential backoff and jitter, or after the delay in Retry-After.
// A POST request is only retried when it carries an idempotency key
// ([WithIdempotencyKey]), so an email is never sent twice. Quota errors
// (daily_quota_exceeded, monthly_quota_exceeded) are never retried.
func WithRetries(n int) Option {
	return func(c *Client) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

// WithTimeout sets the timeout of each attempt (default 30s, 0 disables).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d >= 0 {
			c.timeout = d
		}
	}
}

// WithUserAgent appends a product token to the pidgely-go/<version> user agent.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// WithHeader adds a header sent with every request.
func WithHeader(key, value string) Option {
	return func(c *Client) { c.headers.Add(key, value) }
}

// NewClient creates a client. When apiKey is empty, the PIDGELY_API_KEY
// environment variable is used; requests made without any key fail with an
// [Error] named "missing_api_key".
func NewClient(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("PIDGELY_API_KEY")
	}
	baseURL := os.Getenv("PIDGELY_BASE_URL")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	c := &Client{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
		timeout:    defaultTimeout,
		maxRetries: defaultMaxRetries,
		headers:    http.Header{},
		sleep:      sleepContext,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.Emails = &EmailsService{client: c, Attachments: &EmailAttachmentsService{client: c}}
	c.Batch = &BatchService{client: c}
	c.Domains = &DomainsService{client: c}
	c.ApiKeys = &ApiKeysService{client: c}
	c.Contacts = &ContactsService{client: c, Segments: &ContactSegmentsService{client: c}}
	c.Segments = &SegmentsService{client: c}
	c.Audiences = c.Segments
	c.Broadcasts = &BroadcastsService{client: c}
	c.Templates = &TemplatesService{client: c}
	c.Webhooks = &WebhooksService{client: c}
	c.Suppressions = &SuppressionsService{client: c, Batch: &SuppressionsBatchService{client: c}}
	return c
}

// BaseURL returns the API base URL in use.
func (c *Client) BaseURL() string { return c.baseURL }

// String describes the client without its API key.
func (c *Client) String() string {
	return fmt.Sprintf("pidgely.Client{BaseURL: %q, ApiKey: %q}", c.baseURL, redacted(c.apiKey))
}

// GoString implements fmt.GoStringer so %#v does not print the API key.
func (c *Client) GoString() string { return c.String() }

// LogValue implements slog.LogValuer so structured logs do not include the API key.
func (c *Client) LogValue() slog.Value {
	return slog.GroupValue(slog.String("base_url", c.baseURL), slog.String("api_key", redacted(c.apiKey)))
}

func redacted(key string) string {
	if key == "" {
		return ""
	}
	return "[REDACTED]"
}

// Bool returns a pointer to v, for optional boolean fields such as
// UpdateContactRequest.Unsubscribed.
func Bool(v bool) *bool { return &v }

// String returns a pointer to v.
func String(v string) *string { return &v }

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
