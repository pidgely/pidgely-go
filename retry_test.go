package pidgely

import (
	"context"
	"testing"
	"time"
)

var (
	rateLimited = reply{status: 429, body: `{"statusCode":429,"name":"rate_limit_exceeded","message":"slow down"}`}
	unavailable = reply{status: 503, body: `{"statusCode":503,"name":"internal_server_error","message":"busy"}`}
)

func TestRetriesGETOn5xx(t *testing.T) {
	api, c := newAPI(t, unavailable, unavailable, reply{body: `{"id":"e1"}`})
	e, err := c.Emails.Get(context.Background(), "e1")
	if err != nil || e.Id != "e1" {
		t.Fatalf("e=%v err=%v", e, err)
	}
	if api.count() != 3 || len(api.slept()) != 2 {
		t.Fatalf("calls=%d sleeps=%v", api.count(), api.slept())
	}
	if api.slept()[0] < 250*time.Millisecond || api.slept()[0] > 500*time.Millisecond {
		t.Fatalf("first backoff %v", api.slept()[0])
	}
	if api.slept()[1] < 500*time.Millisecond || api.slept()[1] > time.Second {
		t.Fatalf("second backoff %v", api.slept()[1])
	}
}

func TestStopsAfterMaxRetries(t *testing.T) {
	api, c := newAPI(t, unavailable, unavailable, unavailable, unavailable)
	_, err := c.Domains.List(context.Background(), nil)
	if e, ok := AsError(err); !ok || e.StatusCode != 503 {
		t.Fatalf("err = %v", err)
	}
	if api.count() != 3 {
		t.Fatalf("calls = %d", api.count())
	}
}

func TestRetriesConfigurable(t *testing.T) {
	api, c := newAPI(t, unavailable, unavailable)
	c.maxRetries = 0
	if _, err := c.Domains.List(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
	if api.count() != 1 {
		t.Fatalf("calls = %d", api.count())
	}
	if NewClient("k", WithRetries(5)).maxRetries != 5 || NewClient("k", WithRetries(-1)).maxRetries != 2 {
		t.Fatal("WithRetries")
	}
	if NewClient("k", WithTimeout(-1)).timeout != 30*time.Second {
		t.Fatal("WithTimeout negative ignored")
	}
}

func TestPerRequestRetries(t *testing.T) {
	api, c := newAPI(t, unavailable, unavailable, unavailable, unavailable, unavailable)
	_, _ = c.Domains.List(context.Background(), nil, WithRequestRetries(4))
	if api.count() != 5 {
		t.Fatalf("calls = %d", api.count())
	}
}

func TestNeverRetriesPOSTWithoutIdempotencyKey(t *testing.T) {
	for name, r := range map[string]reply{"503": unavailable, "429": rateLimited, "500": {status: 500, body: "oops"}} {
		t.Run(name, func(t *testing.T) {
			api, c := newAPI(t, r, reply{body: `{"id":"dup"}`})
			_, err := c.Emails.Send(context.Background(), &SendEmailRequest{To: []string{"a@b.c"}})
			if err == nil {
				t.Fatal("expected the error to be returned")
			}
			if api.count() != 1 {
				t.Fatalf("POST without idempotency key was retried (%d calls)", api.count())
			}
		})
	}
}

func TestNeverRetriesPOSTWithoutKeyOnNetworkError(t *testing.T) {
	api, c := newAPI(t)
	api.server.Close()
	_, err := c.Emails.Send(context.Background(), &SendEmailRequest{To: []string{"a@b.c"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(api.slept()) != 0 {
		t.Fatal("POST without idempotency key must not be retried after a network error")
	}
}

func TestRetriesPOSTWithIdempotencyKey(t *testing.T) {
	api, c := newAPI(t, unavailable, rateLimited, reply{body: `{"id":"e1"}`})
	res, err := c.Emails.Send(context.Background(), &SendEmailRequest{To: []string{"a@b.c"}}, WithIdempotencyKey("k1"))
	if err != nil || res.Id != "e1" {
		t.Fatalf("res=%v err=%v", res, err)
	}
	if api.count() != 3 {
		t.Fatalf("calls = %d", api.count())
	}
	for _, call := range api.all() {
		if call.Header.Get("Idempotency-Key") != "k1" {
			t.Fatal("every attempt must carry the same key")
		}
		mustEqualJSON(t, call.JSON(t), `{"to":["a@b.c"]}`)
	}
}

func TestRetriesBatchWithIdempotencyKey(t *testing.T) {
	api, c := newAPI(t, unavailable, reply{body: `{"data":[{"id":"a"}]}`})
	res, err := c.Batch.Send(context.Background(), []SendEmailRequest{{To: []string{"a@b.c"}}}, WithIdempotencyKey("b1"))
	if err != nil || res.Data[0].Id != "a" || api.count() != 2 {
		t.Fatalf("res=%v err=%v calls=%d", res, err, api.count())
	}
}

func TestRetriesPATCHAndDELETE(t *testing.T) {
	api, c := newAPI(t, unavailable, reply{body: `{"object":"domain","id":"d"}`}, unavailable, reply{body: `{"deleted":true}`})
	if _, err := c.Domains.Update(context.Background(), "d", &UpdateDomainRequest{Tls: TLSEnforced}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Domains.Remove(context.Background(), "d"); err != nil {
		t.Fatal(err)
	}
	if api.count() != 4 {
		t.Fatalf("calls = %d", api.count())
	}
}

func TestHonoursRetryAfter(t *testing.T) {
	api, c := newAPI(t, reply{status: 429, body: `{"name":"rate_limit_exceeded","message":"x"}`, headers: map[string]string{"Retry-After": "3"}}, reply{body: `{}`})
	if _, err := c.Domains.List(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if len(api.slept()) != 1 || api.slept()[0] != 3*time.Second {
		t.Fatalf("sleeps = %v", api.slept())
	}
}

func TestRetryAfterTooLong(t *testing.T) {
	api, c := newAPI(t, reply{status: 429, body: `{"name":"rate_limit_exceeded","message":"x"}`, headers: map[string]string{"Retry-After": "120"}})
	_, err := c.Domains.List(context.Background(), nil)
	if !IsRateLimited(err) || api.count() != 1 {
		t.Fatalf("err=%v calls=%d", err, api.count())
	}
}

func TestQuotaErrorsNotRetried(t *testing.T) {
	api, c := newAPI(t, reply{status: 429, body: `{"statusCode":429,"name":"daily_quota_exceeded","message":"quota"}`})
	_, err := c.Domains.List(context.Background(), nil)
	if e, _ := AsError(err); e == nil || e.Name != ErrDailyQuotaExceeded || api.count() != 1 {
		t.Fatalf("err=%v calls=%d", err, api.count())
	}
}

func TestNonRetryableStatus(t *testing.T) {
	api, c := newAPI(t, reply{status: 422, body: `{"name":"validation_error","message":"bad"}`})
	_, _ = c.Domains.List(context.Background(), nil)
	if api.count() != 1 {
		t.Fatalf("calls = %d", api.count())
	}
}

func TestRetriesGETOnNetworkError(t *testing.T) {
	api, c := newAPI(t)
	api.server.Close()
	_, err := c.Domains.List(context.Background(), nil)
	if err == nil || len(api.slept()) != 2 {
		t.Fatalf("err=%v sleeps=%v", err, api.slept())
	}
}

func TestCancelDuringBackoff(t *testing.T) {
	api, c := newAPI(t, unavailable, reply{body: `{}`})
	ctx, cancel := context.WithCancel(context.Background())
	c.sleep = func(ctx context.Context, _ time.Duration) error {
		cancel()
		return ctx.Err()
	}
	_, err := c.Domains.List(ctx, nil)
	if err == nil || api.count() != 1 {
		t.Fatalf("err=%v calls=%d", err, api.count())
	}
}

func TestSleepContext(t *testing.T) {
	if err := sleepContext(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepContext(ctx, time.Hour); err == nil {
		t.Fatal("expected cancellation")
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	cases := map[string]struct {
		d  time.Duration
		ok bool
	}{
		"":                              {0, false},
		"2":                             {2 * time.Second, true},
		"1.5":                           {1500 * time.Millisecond, true},
		"-1":                            {0, false},
		"soon":                          {0, false},
		"Mon, 12 Oct 2026 12:00:05 GMT": {5 * time.Second, true},
		"Mon, 12 Oct 2026 11:00:00 GMT": {0, true},
	}
	for in, want := range cases {
		d, ok := parseRetryAfter(in, now)
		if d != want.d || ok != want.ok {
			t.Errorf("parseRetryAfter(%q) = %v,%v want %v,%v", in, d, ok, want.d, want.ok)
		}
	}
}

func TestBackoffBounds(t *testing.T) {
	for attempt := 0; attempt < 10; attempt++ {
		exp := min(backoffCap, backoffBase<<min(attempt, 5))
		for i := 0; i < 50; i++ {
			d := backoff(attempt)
			if d < exp/2 || d > exp {
				t.Fatalf("backoff(%d) = %v outside [%v, %v]", attempt, d, exp/2, exp)
			}
		}
	}
}
