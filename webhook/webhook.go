// Package webhook verifies Pidgely webhook deliveries.
//
// Pidgely signs every delivery with the Standard Webhooks scheme
// (https://www.standardwebhooks.com), the same one Svix uses. The values are sent
// under both the webhook-* and svix-* header names. Verify checks the
// HMAC-SHA256 signature in constant time and rejects timestamps more than five
// minutes away from the current time.
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//		payload, _ := io.ReadAll(r.Body)
//		event, err := webhook.Parse(payload, r.Header, os.Getenv("PIDGELY_WEBHOOK_SECRET"))
//		if err != nil {
//			http.Error(w, "invalid signature", http.StatusBadRequest)
//			return
//		}
//		// event.Type, event.Data …
//	}
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultTolerance is the maximum accepted clock difference.
const DefaultTolerance = 5 * time.Minute

// Verification errors. Every error returned by Verify and Parse wraps ErrVerification.
var (
	ErrVerification       = errors.New("webhook verification failed")
	ErrMissingSecret      = errors.New("missing webhook secret")
	ErrInvalidSecret      = errors.New("invalid webhook secret")
	ErrMissingHeaders     = errors.New("missing required headers")
	ErrInvalidHeaders     = errors.New("invalid signature headers")
	ErrTimestampTooOld    = errors.New("message timestamp too old")
	ErrTimestampTooNew    = errors.New("message timestamp too new")
	ErrNoMatchingSig      = errors.New("no matching signature found")
	ErrInvalidJSONPayload = errors.New("payload is not valid JSON")
)

type verificationError struct{ err error }

func (e *verificationError) Error() string { return "pidgely webhook: " + e.err.Error() }
func (e *verificationError) Unwrap() []error {
	return []error{e.err, ErrVerification}
}

func fail(err error) error { return &verificationError{err: err} }

type config struct {
	tolerance time.Duration
	now       func() time.Time
}

// Option configures verification.
type Option func(*config)

// WithTolerance sets the maximum accepted clock difference (default 5 minutes).
func WithTolerance(d time.Duration) Option {
	return func(c *config) { c.tolerance = d }
}

// WithNow overrides the current time (useful in tests).
func WithNow(now func() time.Time) Option {
	return func(c *config) { c.now = now }
}

// Event is a webhook delivery payload (Resend format).
type Event struct {
	// Type is the event type, such as "email.delivered".
	Type string `json:"type"`
	// CreatedAt is when the event happened.
	CreatedAt string `json:"created_at"`
	// Data is the event-specific object; decode it with json.Unmarshal.
	Data json.RawMessage `json:"data"`
}

func header(h http.Header, name string) string {
	if v := h.Values("Webhook-" + name); len(v) > 0 {
		return strings.Join(v, " ")
	}
	return strings.Join(h.Values("Svix-"+name), " ")
}

// Sign computes the "v1,<base64>" signature of a delivery.
func Sign(secret, msgID string, timestamp int64, payload []byte) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	return sign(key, msgID, strconv.FormatInt(timestamp, 10), payload), nil
}

func decodeSecret(secret string) ([]byte, error) {
	if secret == "" {
		return nil, fail(ErrMissingSecret)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil || len(key) == 0 {
		return nil, fail(ErrInvalidSecret)
	}
	return key, nil
}

func sign(key []byte, msgID, timestamp string, payload []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msgID + "." + timestamp + "."))
	mac.Write(payload)
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Verify checks the signature of a delivery. payload must be the raw request
// body, exactly as received. Both webhook-* and svix-* headers are accepted.
// Errors wrap ErrVerification and a specific cause such as ErrNoMatchingSig.
func Verify(payload []byte, headers http.Header, secret string, opts ...Option) error {
	cfg := config{tolerance: DefaultTolerance, now: time.Now}
	for _, opt := range opts {
		opt(&cfg)
	}
	key, err := decodeSecret(secret)
	if err != nil {
		return err
	}
	msgID, ts, sigs := header(headers, "Id"), header(headers, "Timestamp"), header(headers, "Signature")
	if msgID == "" || ts == "" || sigs == "" {
		return fail(ErrMissingHeaders)
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || sec < 0 {
		return fail(ErrInvalidHeaders)
	}
	now := cfg.now().Unix()
	tol := int64(cfg.tolerance / time.Second)
	if now-sec > tol {
		return fail(ErrTimestampTooOld)
	}
	if sec-now > tol {
		return fail(ErrTimestampTooNew)
	}

	expected := []byte(sign(key, msgID, ts, payload))
	ok := false
	for _, candidate := range strings.Fields(sigs) {
		// Compare every candidate so timing does not reveal which one matched.
		if hmac.Equal([]byte(candidate), expected) {
			ok = true
		}
	}
	if !ok {
		return fail(ErrNoMatchingSig)
	}
	return nil
}

// Parse verifies a delivery with [Verify] and decodes it into an [Event].
func Parse(payload []byte, headers http.Header, secret string, opts ...Option) (*Event, error) {
	if err := Verify(payload, headers, secret, opts...); err != nil {
		return nil, err
	}
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fail(ErrInvalidJSONPayload)
	}
	return &event, nil
}
