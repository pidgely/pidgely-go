# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-10-12

### Added

- First release of the official Pidgely SDK for Go.
- `pidgely.NewClient` that reads `PIDGELY_API_KEY` / `PIDGELY_BASE_URL`, with the options `WithBaseURL`, `WithHTTPClient`, `WithRetries`, `WithTimeout`, `WithUserAgent` and `WithHeader`. Standard library only, no dependencies.
- Typed request and response structs for every resource: `Emails` (with attachments), `Batch`, `Domains`, `APIKeys`, `Contacts` (with `Segments`), `Segments` (with the `Audiences` alias), `Broadcasts`, `Templates`, `Webhooks` and `Suppressions` (with `Batch`). Every method takes a `context.Context`.
- `*pidgely.Error` with `StatusCode`, `Name`, `Message` and `Headers` for API, network and timeout failures, plus `IsNotFound` and `IsRateLimited`.
- Per-request options: `WithIdempotencyKey` (`Idempotency-Key` header), `WithBatchValidation`, `WithRequestHeader`, `WithRequestRetries` and `WithRequestTimeout`.
- Automatic retries with exponential backoff and jitter on 408/429/5xx/network errors, honouring `Retry-After`. A POST is only retried when it carries an idempotency key, so an email is never sent twice.
- Per-attempt timeouts and the `pidgely-go/<version>` user agent. The API key is redacted from the client's `String`, `GoString` and `slog` output and never appears in errors.
- `webhook.Verify`, `webhook.Parse` and `webhook.Sign` for Standard Webhooks (Svix) signatures, with constant-time comparison and a 5-minute timestamp tolerance.

[Unreleased]: https://github.com/pidgely/pidgely-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/pidgely/pidgely-go/releases/tag/v0.1.0
