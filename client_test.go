package pidgely

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSendEmailRequest(t *testing.T) {
	api, c := newAPI(t, reply{body: `{"id":"e1"}`})
	res, err := c.Emails.Send(context.Background(), &SendEmailRequest{
		From:        "Acme <hello@acme.com>",
		To:          []string{"ana@example.org"},
		Subject:     "Welcome",
		Html:        "<p>Hi</p>",
		ReplyTo:     []string{"support@acme.com"},
		Attachments: []Attachment{{Filename: "a.txt", Content: []byte("hello")}, {Filename: "logo.png", Path: "https://acme.com/logo.png", ContentId: "logo"}},
		Tags:        []Tag{{Name: "category", Value: "billing"}},
		Headers:     map[string]string{"X-Entity-Ref-Id": "1"},
		ScheduledAt: "in 1 hour",
	}, WithIdempotencyKey("welcome-ana"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Id != "e1" {
		t.Fatalf("id = %q", res.Id)
	}
	r := api.last()
	if r.Method != "POST" || r.Path != "/emails" {
		t.Fatalf("%s %s", r.Method, r.Path)
	}
	if got := r.Header.Get("Authorization"); got != "Bearer "+testKey {
		t.Fatalf("Authorization = %q", got)
	}
	if got := r.Header.Get("User-Agent"); got != "pidgely-go/"+Version {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := r.Header.Get("Idempotency-Key"); got != "welcome-ana" {
		t.Fatalf("Idempotency-Key = %q", got)
	}
	mustEqualJSON(t, r.JSON(t), `{
		"from":"Acme <hello@acme.com>","to":["ana@example.org"],"reply_to":["support@acme.com"],
		"subject":"Welcome","html":"<p>Hi</p>","headers":{"X-Entity-Ref-Id":"1"},
		"attachments":[{"filename":"a.txt","content":"aGVsbG8="},{"filename":"logo.png","path":"https://acme.com/logo.png","content_id":"logo"}],
		"tags":[{"name":"category","value":"billing"}],"scheduled_at":"in 1 hour"}`)
}

func TestSendTemplate(t *testing.T) {
	api, c := newAPI(t, reply{body: `{"id":"e2"}`})
	_, err := c.Emails.Send(context.Background(), &SendEmailRequest{
		From: "a@acme.com", To: []string{"b@example.org"},
		Template: &TemplateRef{Id: "welcome", Variables: map[string]any{"NAME": "Ana", "DAYS": 14}},
		Stream:   StreamBroadcast,
	})
	if err != nil {
		t.Fatal(err)
	}
	mustEqualJSON(t, api.last().JSON(t), `{"from":"a@acme.com","to":["b@example.org"],"template":{"id":"welcome","variables":{"DAYS":14,"NAME":"Ana"}},"stream":"broadcast"}`)
	if api.last().Header.Get("Idempotency-Key") != "" {
		t.Fatal("unexpected Idempotency-Key")
	}
}

func TestDecodesResponses(t *testing.T) {
	_, c := newAPI(t, reply{body: `{"object":"email","id":"e1","to":["a@b.c"],"from":"x@y.z","subject":"s","html":null,"created_at":"2026-10-12 16:44:41.000000+00","scheduled_at":null,"last_event":"delivered","tags":[{"name":"n","value":"v"}]}`})
	e, err := c.Emails.Get(context.Background(), "e1")
	if err != nil {
		t.Fatal(err)
	}
	if e.LastEvent != "delivered" || e.To[0] != "a@b.c" || e.Html != "" || e.Tags[0].Value != "v" {
		t.Fatalf("%+v", e)
	}
}

func TestUserAgentAndHeaders(t *testing.T) {
	api, c := newAPI(t)
	c2 := NewClient(testKey, WithBaseURL(c.BaseURL()), WithUserAgent("my-app/1.2"), WithHeader("X-Team", "growth"))
	if _, err := c2.Domains.List(context.Background(), nil, WithRequestHeader("X-Trace", "abc")); err != nil {
		t.Fatal(err)
	}
	r := api.last()
	if got := r.Header.Get("User-Agent"); got != "pidgely-go/"+Version+" my-app/1.2" {
		t.Fatalf("User-Agent = %q", got)
	}
	if r.Header.Get("X-Team") != "growth" || r.Header.Get("X-Trace") != "abc" {
		t.Fatalf("headers = %v", r.Header)
	}
	if r.Header.Get("Content-Type") != "" {
		t.Fatal("GET must not send Content-Type")
	}
}

func TestEnvironment(t *testing.T) {
	t.Setenv("PIDGELY_API_KEY", "pdg_env_key")
	t.Setenv("PIDGELY_BASE_URL", "https://example.test/")
	c := NewClient("")
	if c.apiKey != "pdg_env_key" || c.BaseURL() != "https://example.test" {
		t.Fatalf("apiKey/baseURL from env not used: %s", c.BaseURL())
	}
	if NewClient("pdg_explicit").apiKey != "pdg_explicit" {
		t.Fatal("explicit key must win")
	}
	if NewClient("x", WithBaseURL("https://other.test")).BaseURL() != "https://other.test" {
		t.Fatal("WithBaseURL must win")
	}
	t.Setenv("PIDGELY_BASE_URL", "")
	if NewClient("x").BaseURL() != DefaultBaseURL {
		t.Fatal("default base URL")
	}
}

func TestMissingAPIKey(t *testing.T) {
	t.Setenv("PIDGELY_API_KEY", "")
	_, err := NewClient("").Emails.Get(context.Background(), "x")
	e, ok := AsError(err)
	if !ok || e.Name != ErrMissingAPIKey || e.StatusCode != 0 {
		t.Fatalf("err = %v", err)
	}
}

func TestAPIKeyNeverLeaks(t *testing.T) {
	api, c := newAPI(t,
		reply{status: 401, body: `{"statusCode":401,"name":"invalid_api_key","message":"API key is invalid"}`},
	)
	var outputs []string
	outputs = append(outputs, fmt.Sprintf("%v", c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c), fmt.Sprint(c), c.String())
	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("client", "client", c)
	outputs = append(outputs, buf.String())

	_, err := c.Emails.Get(context.Background(), "e")
	outputs = append(outputs, err.Error(), fmt.Sprintf("%+v", err), fmt.Sprintf("%#v", err))

	// Network error and timeout messages.
	api.server.Close()
	_, err = c.Emails.Get(context.Background(), "e", WithRequestRetries(0))
	outputs = append(outputs, err.Error(), fmt.Sprintf("%#v", err))

	for _, out := range outputs {
		if strings.Contains(out, testKey) || strings.Contains(out, "0123456789abcdef") {
			t.Fatalf("API key leaked in %q", out)
		}
	}
	if !strings.Contains(c.String(), "[REDACTED]") {
		t.Fatalf("String() = %q", c.String())
	}
}

func TestSanitize(t *testing.T) {
	if got := sanitize("bad "+testKey+" here", testKey); strings.Contains(got, testKey) {
		t.Fatal(got)
	}
	if sanitize("x", "") != "x" {
		t.Fatal("empty key")
	}
}

func TestAPIErrors(t *testing.T) {
	cases := []struct {
		name  string
		reply reply
		want  Error
	}{
		{"resend shape", reply{status: 422, body: `{"statusCode":422,"name":"validation_error","message":"The to field is missing."}`}, Error{StatusCode: 422, Name: "validation_error", Message: "The to field is missing."}},
		{"no statusCode", reply{status: 404, body: `{"name":"not_found","message":"nope"}`}, Error{StatusCode: 404, Name: "not_found", Message: "nope"}},
		{"native shape", reply{status: 403, body: `{"error":{"code":"forbidden","message":"no"}}`}, Error{StatusCode: 403, Name: "forbidden", Message: "no"}},
		{"native without message", reply{status: 403, body: `{"error":{"code":"forbidden"}}`}, Error{StatusCode: 403, Name: "forbidden", Message: "forbidden"}},
		{"html 5xx", reply{status: 501, body: "<html>Bad</html>"}, Error{StatusCode: 501, Name: "internal_server_error", Message: "<html>Bad</html>"}},
		{"empty 4xx", reply{status: 400}, Error{StatusCode: 400, Name: "application_error", Message: "HTTP 400"}},
		{"json array", reply{status: 400, body: `[1]`}, Error{StatusCode: 400, Name: "application_error", Message: "[1]"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, c := newAPI(t, tc.reply)
			_, err := c.Domains.Get(context.Background(), "d")
			e, ok := AsError(err)
			if !ok {
				t.Fatalf("err = %v", err)
			}
			if e.StatusCode != tc.want.StatusCode || e.Name != tc.want.Name || e.Message != tc.want.Message {
				t.Fatalf("got %+v, want %+v", *e, tc.want)
			}
			if e.Headers == nil {
				t.Fatal("headers missing")
			}
		})
	}
}

func TestErrorHelpers(t *testing.T) {
	_, c := newAPI(t, reply{status: 404, body: `{"statusCode":404,"name":"not_found","message":"Email not found"}`})
	_, err := c.Emails.Get(context.Background(), "x")
	if !IsNotFound(err) || IsRateLimited(err) {
		t.Fatal("IsNotFound")
	}
	if err.Error() != "pidgely: not_found (HTTP 404): Email not found" {
		t.Fatalf("Error() = %q", err.Error())
	}
	wrapped := fmt.Errorf("context: %w", err)
	if !IsNotFound(wrapped) {
		t.Fatal("wrapped")
	}
	if IsNotFound(errors.New("other")) {
		t.Fatal("plain error")
	}
	e := &Error{Name: ErrApplication, Message: "boom"}
	if e.Error() != "pidgely: application_error: boom" {
		t.Fatal(e.Error())
	}
}

func TestInvalidSuccessJSON(t *testing.T) {
	_, c := newAPI(t, reply{body: "{not json"})
	_, err := c.Emails.Get(context.Background(), "x")
	if e, ok := AsError(err); !ok || e.Name != ErrApplication || e.Message != "unable to parse the API response" {
		t.Fatalf("err = %v", err)
	}
}

func TestEmptySuccessBody(t *testing.T) {
	_, c := newAPI(t, reply{body: ""})
	d, err := c.Domains.Remove(context.Background(), "d")
	if err != nil || d == nil || d.Id != "" {
		t.Fatalf("d=%v err=%v", d, err)
	}
}

func TestRequiredArguments(t *testing.T) {
	api, c := newAPI(t)
	_, err := c.Emails.Get(context.Background(), "")
	if e, ok := AsError(err); !ok || e.Name != ErrMissingRequiredField {
		t.Fatalf("err = %v", err)
	}
	if api.count() != 0 {
		t.Fatal("no request expected")
	}
}

func TestTimeout(t *testing.T) {
	api, c := newAPI(t, reply{hang: true})
	_, err := c.Emails.Get(context.Background(), "x", WithRequestTimeout(30*time.Millisecond), WithRequestRetries(0))
	e, ok := AsError(err)
	if !ok || e.Name != ErrApplication || e.StatusCode != 0 || !strings.Contains(e.Message, "timed out after 30ms") {
		t.Fatalf("err = %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("should wrap DeadlineExceeded")
	}
	if api.count() != 1 {
		t.Fatal(api.count())
	}
}

func TestClientTimeoutOption(t *testing.T) {
	api, _ := newAPI(t, reply{hang: true})
	c := NewClient(testKey, WithBaseURL(api.server.URL), WithTimeout(20*time.Millisecond), WithRetries(0), WithHTTPClient(&http.Client{}))
	_, err := c.Emails.Get(context.Background(), "x")
	if e, ok := AsError(err); !ok || !strings.Contains(e.Message, "timed out after 20ms") {
		t.Fatalf("err = %v", err)
	}
}

func TestCallerCancel(t *testing.T) {
	_, c := newAPI(t, reply{hang: true})
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	_, err := c.Emails.Get(ctx, "x")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestNetworkError(t *testing.T) {
	api, c := newAPI(t)
	api.server.Close()
	_, err := c.Emails.Get(context.Background(), "x", WithRequestRetries(0))
	e, ok := AsError(err)
	if !ok || e.Name != ErrApplication || e.StatusCode != 0 || !strings.HasPrefix(e.Message, "unable to reach the Pidgely API") || e.Headers != nil {
		t.Fatalf("err = %#v", err)
	}
	if e.Unwrap() == nil {
		t.Fatal("cause missing")
	}
}

func TestNilContext(t *testing.T) {
	api, c := newAPI(t)
	var ctx context.Context // deliberately nil
	if _, err := c.Domains.List(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if api.count() != 1 {
		t.Fatal("request not sent")
	}
}
