<p align="center">
  <a href="https://pidgely.com"><img src="https://raw.githubusercontent.com/pidgely/.github/main/assets/banner.png" alt="Pidgely — Email API for developers" width="100%"></a>
</p>

<p align="center">
  <a href="https://github.com/pidgely/pidgely-go/actions/workflows/ci.yml"><img src="https://github.com/pidgely/pidgely-go/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://pkg.go.dev/github.com/pidgely/pidgely-go"><img src="https://pkg.go.dev/badge/github.com/pidgely/pidgely-go.svg" alt="Go Reference"></a>
  <img src="https://img.shields.io/badge/go-%E2%89%A51.23-00ADD8?logo=go&logoColor=white" alt="Go ≥ 1.23">
  <img src="https://img.shields.io/badge/dependencies-0-success" alt="Zero dependencies">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT"></a>
</p>

# pidgely-go

The official Go SDK for [Pidgely](https://pidgely.com), the Resend-compatible email API.

- **Drop-in for Resend.** The same services, method names and request structs as `resend-go`. Migrating means changing an import and a key.
- **Standard library only.** No dependencies. Every method takes a `context.Context`.
- **Typed.** Request and response structs for every resource.
- **Built for production.** Idempotency keys, automatic retries with backoff that honour `Retry-After`, per-attempt timeouts.
- **Webhook verification.** Standard Webhooks (Svix) signatures, checked in constant time with timestamp tolerance.

Full documentation lives at **[docs.pidgely.com](https://docs.pidgely.com)**.

## Install

```sh
go get github.com/pidgely/pidgely-go
```

Requires Go 1.23 or later.

## Quickstart

Create an API key in the [dashboard](https://app.pidgely.com), verify a sending domain, then:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pidgely/pidgely-go"
)

func main() {
	ctx := context.Background()
	client := pidgely.NewClient(os.Getenv("PIDGELY_API_KEY"))

	sent, err := client.Emails.Send(ctx, &pidgely.SendEmailRequest{
		From:    "Acme <hello@acme.com>",
		To:      []string{"ana@example.org"},
		Subject: "Welcome to Acme",
		Html:    "<p>Your account is ready.</p>",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sent.Id)
}
```

### Configuration

```go
client := pidgely.NewClient(os.Getenv("PIDGELY_API_KEY"),
	pidgely.WithBaseURL("https://api.pidgely.com"), // or PIDGELY_BASE_URL
	pidgely.WithTimeout(30*time.Second),           // per attempt (0 disables)
	pidgely.WithRetries(2),                        // automatic retries, see below
	pidgely.WithHTTPClient(&http.Client{}),        // custom transport, proxies, tracing
	pidgely.WithUserAgent("my-app/1.2.3"),         // appended to pidgely-go/<version>
	pidgely.WithHeader("X-Team", "growth"),        // sent with every request
)
```

| Environment variable | Used for |
|---|---|
| `PIDGELY_API_KEY` | API key when `NewClient` gets an empty string |
| `PIDGELY_BASE_URL` | Base URL when `WithBaseURL` is not set (default `https://api.pidgely.com`) |

The client is safe for concurrent use. Its `String`, `GoString` and `slog` representations never include the API key.

### Errors

Every failure is a `*pidgely.Error`:

```go
sent, err := client.Emails.Send(ctx, req)
var apiErr *pidgely.Error
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode, apiErr.Name, apiErr.Message) // 422 validation_error ...
}
```

`Name` is machine-readable: `validation_error`, `not_found`, `invalid_api_key`, `restricted_api_key`, `rate_limit_exceeded`, `daily_quota_exceeded`, `invalid_idempotent_request` and so on (constants `pidgely.ErrValidation`, `pidgely.ErrNotFound`, ...). Network failures, timeouts and cancelled contexts have `StatusCode` 0 and `Name` `application_error`, and wrap the cause, so `errors.Is(err, context.DeadlineExceeded)` works. `pidgely.IsNotFound(err)` and `pidgely.IsRateLimited(err)` are shortcuts.

Every method also accepts request options: `WithIdempotencyKey`, `WithRequestHeader`, `WithRequestTimeout` and `WithRequestRetries`.

### Idempotency

```go
client.Emails.Send(ctx, req, pidgely.WithIdempotencyKey("receipt-1042"))
```

The key is sent as `Idempotency-Key`. For 24 hours the API replays the stored response for the same key (the reply carries the header `idempotent-replayed: true`) and answers `409` when the body differs.

### Retries

Failed requests are retried up to `WithRetries` times (default 2) with exponential backoff and jitter (0.5 s, 1 s, 2 s and so on, capped at 8 s), or after the delay given in `Retry-After`:

| Failure | Retried when |
|---|---|
| `408`, `429`, `500`, `502`, `503`, `504`, network error, timeout | `GET`, `PATCH`, `DELETE`, or a `POST` that has an idempotency key |

**A `POST` without an idempotency key is never retried**, whatever the failure, because the email might already have been accepted. Pass `WithIdempotencyKey` to make sends safe to retry. Quota errors (`daily_quota_exceeded`, `monthly_quota_exceeded`) are never retried, and a `Retry-After` longer than 60 seconds returns the error instead of waiting.

## Resources

All examples assume `client := pidgely.NewClient(os.Getenv("PIDGELY_API_KEY"))` and a `ctx`. List methods take `*pidgely.ListParams{Limit, After, Before}` (cursor pagination; `nil` for defaults) and return `*pidgely.List[T]{Object, HasMore, Data}`.

### Emails

```go
sent, err := client.Emails.Send(ctx, &pidgely.SendEmailRequest{
	From:    "Acme <hello@acme.com>",
	To:      []string{"ana@example.com"},
	Cc:      []string{"team@acme.com"},
	ReplyTo: []string{"support@acme.com"},
	Subject: "Your invoice",
	Html:    "<p>Invoice attached.</p>", // Text is generated when omitted
	Attachments: []pidgely.Attachment{
		{Filename: "invoice.pdf", Content: pdfBytes},
		{Filename: "logo.png", Path: "https://acme.com/logo.png", ContentId: "logo"}, // <img src="cid:logo">
	},
	Tags:        []pidgely.Tag{{Name: "category", Value: "billing"}},
	Headers:     map[string]string{"X-Entity-Ref-ID": "inv_1042"},
	ScheduledAt: "tomorrow at 9am", // ISO 8601 or natural language, up to 30 days ahead
})

// Send a published template
client.Emails.Send(ctx, &pidgely.SendEmailRequest{
	From:     "Acme <hello@acme.com>",
	To:       []string{"ana@example.com"},
	Template: &pidgely.TemplateRef{Id: "welcome", Variables: map[string]any{"NAME": "Ana", "TRIAL_DAYS": 14}},
})

email, err := client.Emails.Get(ctx, "4ef9a417-02e9-4d39-ad75-9611e0fcc33c") // includes LastEvent
page, err := client.Emails.List(ctx, &pidgely.ListParams{Limit: 20})
client.Emails.Update(ctx, id, &pidgely.UpdateEmailRequest{ScheduledAt: "in 2 hours"}) // reschedule
client.Emails.Cancel(ctx, id)                                                         // cancel a scheduled email
client.Emails.Attachments.List(ctx, id, nil)
client.Emails.Attachments.Get(ctx, id, attachmentID) // signed DownloadUrl, valid for 1 hour
```

### Batch

```go
res, err := client.Batch.Send(ctx, []pidgely.SendEmailRequest{
	{From: "Acme <hello@acme.com>", To: []string{"ana@example.com"}, Subject: "Hi Ana", Html: "<p>Hi</p>"},
	{From: "Acme <hello@acme.com>", To: []string{"bob@example.com"}, Subject: "Hi Bob", Html: "<p>Hi</p>"},
}, pidgely.WithIdempotencyKey("digest-2026-10-12"), pidgely.WithBatchValidation(pidgely.BatchPermissive))
// res.Data → ids; with BatchPermissive, res.Errors → [{Index, Message}]
```

Up to 100 emails per request. `BatchStrict` (the default) sends all of them or none. `BatchPermissive` sends the valid ones and reports the invalid ones.

### Domains

```go
domain, err := client.Domains.Create(ctx, &pidgely.CreateDomainRequest{Name: "acme.com", CustomReturnPath: "bounces", Tls: pidgely.TLSOpportunistic})
for _, r := range domain.Records { // DKIM TXT, return-path MX + SPF TXT, DMARC
	fmt.Println(r.Type, r.Name, r.Value)
}
client.Domains.Verify(ctx, domain.Id) // check DNS now
client.Domains.Get(ctx, domain.Id)    // Status: not_started | pending | verified | failed
client.Domains.List(ctx, nil)
client.Domains.Update(ctx, domain.Id, &pidgely.UpdateDomainRequest{Tls: pidgely.TLSEnforced})
client.Domains.Remove(ctx, domain.Id)
```

### API keys

```go
key, err := client.ApiKeys.Create(ctx, &pidgely.CreateApiKeyRequest{Name: "Production", Permission: pidgely.PermissionSendingAccess, DomainId: domainID})
fmt.Println(key.Token) // pdg_…, shown only once
client.ApiKeys.List(ctx, nil)
client.ApiKeys.Update(ctx, key.Id, &pidgely.UpdateApiKeyRequest{Name: "Production (web)"})
client.ApiKeys.Remove(ctx, key.Id)
```

### Contacts

Contacts are addressed by id or by email.

```go
client.Contacts.Create(ctx, &pidgely.CreateContactRequest{
	Email:      "ana@example.com",
	FirstName:  "Ana",
	LastName:   "Lima",
	Properties: map[string]any{"plan": "pro", "seats": 5},
	Segments:   []pidgely.Segment{{Id: segmentID}},
})
client.Contacts.List(ctx, &pidgely.ListContactsParams{SegmentId: segmentID, ListParams: pidgely.ListParams{Limit: 100}})
client.Contacts.Get(ctx, "ana@example.com")
client.Contacts.Update(ctx, "ana@example.com", &pidgely.UpdateContactRequest{Unsubscribed: pidgely.Bool(true)}) // kept in sync with the suppression list
client.Contacts.Remove(ctx, "ana@example.com")

client.Contacts.Segments.List(ctx, "ana@example.com", nil)
client.Contacts.Segments.Add(ctx, "ana@example.com", segmentID)
client.Contacts.Segments.Remove(ctx, contactID, segmentID)
```

### Segments (and legacy audiences)

```go
segment, err := client.Segments.Create(ctx, &pidgely.CreateSegmentRequest{Name: "Pro customers"})
client.Segments.List(ctx, nil)
client.Segments.Get(ctx, segment.Id)
client.Segments.Update(ctx, segment.Id, &pidgely.UpdateSegmentRequest{Name: "Pro & Team customers"})
client.Segments.ListContacts(ctx, segment.Id, &pidgely.ListParams{Limit: 50})
client.Segments.Remove(ctx, segment.Id)
```

`client.Audiences` is a deprecated alias of `client.Segments` (with the type aliases `Audience` and `CreateAudienceRequest`), kept for Resend compatibility. `CreateContactRequest`, `UpdateContactRequest` and `ListContactsParams` also take a legacy `AudienceId`.

### Broadcasts

```go
b, err := client.Broadcasts.Create(ctx, &pidgely.CreateBroadcastRequest{
	SegmentId:   segmentID,
	From:        "Acme News <news@acme.com>",
	Subject:     "What’s new in October",
	PreviewText: "Three features you asked for",
	Html:        `<p>Hi {{{FIRST_NAME|there}}},</p>…<a href="{{{PIDGELY_UNSUBSCRIBE_URL}}}">Unsubscribe</a>`,
	Name:        "October newsletter",
})

client.Broadcasts.Send(ctx, b.Id, &pidgely.SendBroadcastRequest{ScheduledAt: "tomorrow at 10am"}) // nil to send now
client.Broadcasts.Cancel(ctx, b.Id)
client.Broadcasts.Duplicate(ctx, b.Id)
client.Broadcasts.Update(ctx, b.Id, &pidgely.UpdateBroadcastRequest{Subject: "New subject"}) // drafts only
client.Broadcasts.Get(ctx, b.Id) // includes per-status Stats
client.Broadcasts.List(ctx, nil)
client.Broadcasts.Remove(ctx, b.Id)
```

Set `Send: true` (with an optional `ScheduledAt`) on `Create` to send without a separate call. Every broadcast message includes RFC 8058 one-click unsubscribe. Personalisation supports `{{{FIRST_NAME|fallback}}}`, `{{{LAST_NAME}}}`, `{{{EMAIL}}}`, `{{{contact.<property>}}}` and `{{{RESEND_UNSUBSCRIBE_URL}}}` / `{{{PIDGELY_UNSUBSCRIBE_URL}}}`.

### Templates

```go
t, err := client.Templates.Create(ctx, &pidgely.CreateTemplateRequest{
	Name:    "Welcome",
	Alias:   "welcome",
	From:    "Acme <hello@acme.com>",
	Subject: "Welcome, {{{NAME}}}!",
	Html:    "<p>Hi {{{NAME}}}, your trial lasts {{{TRIAL_DAYS}}} days.</p>",
	Variables: []pidgely.TemplateVariableInput{
		{Key: "NAME", Type: "string", FallbackValue: "there"},
		{Key: "TRIAL_DAYS", Type: "number", FallbackValue: 14},
	},
})
client.Templates.Publish(ctx, t.Id) // the published version is what TemplateRef sends

client.Templates.Get(ctx, "welcome") // by id or alias (draft)
client.Templates.Update(ctx, "welcome", &pidgely.UpdateTemplateRequest{Subject: "Welcome aboard, {{{NAME}}}!"})
client.Templates.Duplicate(ctx, "welcome")
client.Templates.List(ctx, nil)
client.Templates.Remove(ctx, "welcome")
```

### Webhooks

```go
wh, err := client.Webhooks.Create(ctx, &pidgely.CreateWebhookRequest{
	Endpoint: "https://acme.com/webhooks/pidgely",
	Events:   []string{pidgely.EventEmailDelivered, pidgely.EventEmailBounced, pidgely.EventEmailComplained},
})
fmt.Println(wh.SigningSecret) // whsec_…, store it to verify deliveries

client.Webhooks.List(ctx, nil)
client.Webhooks.Get(ctx, wh.Id)
client.Webhooks.Update(ctx, wh.Id, &pidgely.UpdateWebhookRequest{Status: "disabled"})
client.Webhooks.RotateSigningSecret(ctx, wh.Id)
client.Webhooks.Remove(ctx, wh.Id)
```

Events: `email.sent`, `email.delivered`, `email.delivery_delayed`, `email.bounced`, `email.complained`, `email.failed`, `email.suppressed`, `domain.updated`, `contact.created`, `contact.updated`, `contact.deleted`.

### Suppressions

```go
client.Suppressions.Add(ctx, &pidgely.AddSuppressionRequest{Email: "noreply@example.com"})
client.Suppressions.List(ctx, &pidgely.ListSuppressionsParams{Origin: pidgely.OriginBounce}) // bounce | complaint | manual | unsubscribe
client.Suppressions.Get(ctx, "noreply@example.com") // by id or email
client.Suppressions.Remove(ctx, "noreply@example.com")
client.Suppressions.Batch.Add(ctx, &pidgely.AddSuppressionBatchRequest{Emails: []string{"a@example.com", "b@example.com"}}) // up to 100
client.Suppressions.Batch.Remove(ctx, &pidgely.RemoveSuppressionBatchRequest{Emails: []string{"a@example.com"}})
```

## Verifying webhooks

Pidgely signs every delivery with the [Standard Webhooks](https://www.standardwebhooks.com) scheme. The same values are sent under both the `webhook-*` and `svix-*` header names. The `webhook` package checks the HMAC-SHA256 signature in constant time and rejects timestamps more than 5 minutes from now. Pass the **raw** request body.

```go
import "github.com/pidgely/pidgely-go/webhook"

http.HandleFunc("/webhooks/pidgely", func(w http.ResponseWriter, r *http.Request) {
	payload, _ := io.ReadAll(r.Body)
	event, err := webhook.Parse(payload, r.Header, os.Getenv("PIDGELY_WEBHOOK_SECRET"))
	if err != nil { // errors.Is(err, webhook.ErrVerification)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	if event.Type == pidgely.EventEmailBounced {
		var data struct {
			EmailId string   `json:"email_id"`
			To      []string `json:"to"`
		}
		_ = json.Unmarshal(event.Data, &data)
	}
	w.WriteHeader(http.StatusNoContent)
})
```

`webhook.Verify(payload, headers, secret)` only checks the signature. Set a different window with `webhook.WithTolerance(d)`. `client.Webhooks.Verify` is a shortcut, and no API key is needed to verify.

## Migrating from Resend

Pidgely serves the Resend REST API at the same paths, and this SDK mirrors `resend-go`:

```diff
- import "github.com/resend/resend-go/v2"
- client := resend.NewClient(os.Getenv("RESEND_API_KEY"))
- sent, err := client.Emails.Send(&resend.SendEmailRequest{...})
+ import "github.com/pidgely/pidgely-go"
+ client := pidgely.NewClient(os.Getenv("PIDGELY_API_KEY"))
+ sent, err := client.Emails.Send(ctx, &pidgely.SendEmailRequest{...})
```

1. `go get github.com/pidgely/pidgely-go` and replace the import.
2. Swap `re_…` for your `pdg_…` key, and pass a `context.Context` as the first argument (the `...WithContext` variants are gone: every method takes one).
3. Verify your domain in Pidgely and publish the DNS records it returns.
4. If you verify webhooks, use `webhook.Verify` with the `whsec_…` secret from your Pidgely webhook.

Request structs keep Resend's field names (`Html`, `ReplyTo`, `ScheduledAt`, `Id`). Pidgely additions include `Emails.Attachments`, `ApiKeys.Update`, `Segments.Update/ListContacts`, `Broadcasts.Cancel/Duplicate`, `Suppressions`, the `Stream` field on emails, and built-in retries. Some Resend features are not supported yet: open/click tracking, topics, inbound email, automations, events and logs. See the migration guide at [docs.pidgely.com](https://docs.pidgely.com) for the full list.

## Development

```sh
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
go test -race -cover ./...   # httptest servers, no network
```

### Publishing

Go modules are published by pushing a semver tag; there is no registry upload and no secret. [`release.yml`](.github/workflows/release.yml) runs on `v*` tags: it checks that the tag matches `pidgely.Version`, runs the tests, warms `proxy.golang.org` (so [pkg.go.dev](https://pkg.go.dev/github.com/pidgely/pidgely-go) lists the version) and creates the GitHub release from this changelog.

1. Update `CHANGELOG.md` and `Version` in `pidgely.go`, commit, and wait for CI to pass on `main`.
2. `git tag v0.2.0 && git push origin v0.2.0`.

## Links

- Documentation: https://docs.pidgely.com
- API reference (OpenAPI 3.1): [`openapi/pidgely.yaml`](openapi/pidgely.yaml)
- Other SDKs: [Node.js](https://github.com/pidgely/pidgely-node) · [Python](https://github.com/pidgely/pidgely-python) · [PHP](https://github.com/pidgely/pidgely-php) · [Ruby](https://github.com/pidgely/pidgely-ruby)
- Security: see [SECURITY.md](https://github.com/pidgely/.github/blob/main/SECURITY.md)

## License

[MIT](LICENSE) © Caotec Serviços em Tecnologia Ltda
