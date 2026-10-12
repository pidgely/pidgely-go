package webhook_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pidgely/pidgely-go/webhook"
)

// Official Standard Webhooks / Svix test vector.
const (
	secret    = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"
	msgID     = "msg_p5jXN8AQM9LWM0D4loKWxJek"
	timestamp = int64(1614265330)
	payload   = `{"test": 2432232314}`
	signature = "v1,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE="
)

func at(offset int64) webhook.Option {
	return webhook.WithNow(func() time.Time { return time.Unix(timestamp+offset, 0) })
}

func svixHeaders() http.Header {
	h := http.Header{}
	h.Set("svix-id", msgID)
	h.Set("svix-timestamp", "1614265330")
	h.Set("svix-signature", signature)
	return h
}

func TestSignMatchesVector(t *testing.T) {
	got, err := webhook.Sign(secret, msgID, timestamp, []byte(payload))
	if err != nil || got != signature {
		t.Fatalf("Sign = %q, %v", got, err)
	}
	got, _ = webhook.Sign(secret[len("whsec_"):], msgID, timestamp, []byte(payload))
	if got != signature {
		t.Fatal("secret without whsec_ prefix")
	}
}

func TestVerifySvixHeaders(t *testing.T) {
	if err := webhook.Verify([]byte(payload), svixHeaders(), secret, at(0)); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyWebhookHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("webhook-id", msgID)
	h.Set("webhook-timestamp", "1614265330")
	h.Set("webhook-signature", signature)
	if err := webhook.Verify([]byte(payload), h, secret, at(0)); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyMultipleSignatures(t *testing.T) {
	h := svixHeaders()
	h.Set("svix-signature", "v1,bm90IHRoaXMgb25l "+signature+" v2,other")
	if err := webhook.Verify([]byte(payload), h, secret, at(0)); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyWithinTolerance(t *testing.T) {
	if err := webhook.Verify([]byte(payload), svixHeaders(), secret, at(299)); err != nil {
		t.Fatal(err)
	}
	if err := webhook.Verify([]byte(payload), svixHeaders(), secret, at(-299)); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRejects(t *testing.T) {
	other := "whsec_" + base64.StdEncoding.EncodeToString([]byte("another secret value!!"))
	cases := []struct {
		name    string
		payload string
		headers func(http.Header)
		secret  string
		opts    []webhook.Option
		want    error
	}{
		{"tampered payload", `{"test": 2432232315}`, nil, secret, []webhook.Option{at(0)}, webhook.ErrNoMatchingSig},
		{"wrong secret", payload, nil, other, []webhook.Option{at(0)}, webhook.ErrNoMatchingSig},
		{"wrong id", payload, func(h http.Header) { h.Set("svix-id", "msg_other") }, secret, []webhook.Option{at(0)}, webhook.ErrNoMatchingSig},
		{"old timestamp", payload, nil, secret, []webhook.Option{at(301)}, webhook.ErrTimestampTooOld},
		{"future timestamp", payload, nil, secret, []webhook.Option{at(-301)}, webhook.ErrTimestampTooNew},
		{"custom tolerance", payload, nil, secret, []webhook.Option{at(11), webhook.WithTolerance(10 * time.Second)}, webhook.ErrTimestampTooOld},
		{"non-numeric timestamp", payload, func(h http.Header) { h.Set("svix-timestamp", "12abc") }, secret, []webhook.Option{at(0)}, webhook.ErrInvalidHeaders},
		{"missing headers", payload, func(h http.Header) { h.Del("svix-signature") }, secret, []webhook.Option{at(0)}, webhook.ErrMissingHeaders},
		{"missing secret", payload, nil, "", []webhook.Option{at(0)}, webhook.ErrMissingSecret},
		{"empty secret key", payload, nil, "whsec_", []webhook.Option{at(0)}, webhook.ErrInvalidSecret},
		{"invalid base64 secret", payload, nil, "whsec_!!!", []webhook.Option{at(0)}, webhook.ErrInvalidSecret},
		{"no v1 signature", payload, func(h http.Header) { h.Set("svix-signature", "v1a,abc") }, secret, []webhook.Option{at(0)}, webhook.ErrNoMatchingSig},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := svixHeaders()
			if tc.headers != nil {
				tc.headers(h)
			}
			err := webhook.Verify([]byte(tc.payload), h, tc.secret, tc.opts...)
			if !errors.Is(err, tc.want) || !errors.Is(err, webhook.ErrVerification) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if err.Error() == "" {
				t.Fatal("empty message")
			}
		})
	}
}

func TestParse(t *testing.T) {
	body := `{"type":"email.delivered","created_at":"2026-10-12T16:44:41Z","data":{"email_id":"e1","to":["a@b.c"]}}`
	sig, err := webhook.Sign(secret, "msg_1", timestamp, []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("webhook-id", "msg_1")
	h.Set("webhook-timestamp", "1614265330")
	h.Set("webhook-signature", sig)
	event, err := webhook.Parse([]byte(body), h, secret, at(0))
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != "email.delivered" || string(event.Data) != `{"email_id":"e1","to":["a@b.c"]}` {
		t.Fatalf("%+v", event)
	}

	// Signed but not JSON.
	sig, _ = webhook.Sign(secret, "msg_2", timestamp, []byte("nope"))
	h.Set("webhook-id", "msg_2")
	h.Set("webhook-signature", sig)
	if _, err := webhook.Parse([]byte("nope"), h, secret, at(0)); !errors.Is(err, webhook.ErrInvalidJSONPayload) {
		t.Fatalf("err = %v", err)
	}
	// Bad signature.
	if _, err := webhook.Parse([]byte(body), svixHeaders(), secret, at(0)); !errors.Is(err, webhook.ErrNoMatchingSig) {
		t.Fatalf("err = %v", err)
	}
}

func TestSignInvalidSecret(t *testing.T) {
	if _, err := webhook.Sign("", "id", 1, nil); !errors.Is(err, webhook.ErrMissingSecret) {
		t.Fatal(err)
	}
}

func TestDefaultNow(t *testing.T) {
	// Without WithNow the current time is used: the 2021 vector is too old.
	if err := webhook.Verify([]byte(payload), svixHeaders(), secret); !errors.Is(err, webhook.ErrTimestampTooOld) {
		t.Fatal(err)
	}
}
