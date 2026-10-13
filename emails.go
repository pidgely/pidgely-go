package pidgely

import (
	"context"
	"net/http"
)

// Tag is a name/value pair attached to an email (letters, digits, _ and -).
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Attachment is a file attached to an email. Set either Content or Path.
type Attachment struct {
	// Filename is the name shown to the recipient.
	Filename string `json:"filename,omitempty"`
	// Content is the raw file content (base64-encoded on the wire).
	Content []byte `json:"content,omitempty"`
	// Path is an http(s) URL downloaded at send time.
	Path string `json:"path,omitempty"`
	// ContentType overrides the detected MIME type.
	ContentType string `json:"content_type,omitempty"`
	// ContentId makes the attachment inline; reference it as cid:<ContentId>.
	ContentId string `json:"content_id,omitempty"`
}

// TemplateRef sends a published template instead of Html/Text.
type TemplateRef struct {
	// Id is the template id or alias (its published version is used).
	Id string `json:"id"`
	// Variables fill the template's {{{VARIABLES}}}; values are strings or numbers.
	Variables map[string]any `json:"variables,omitempty"`
}

// Email streams.
const (
	StreamTransactional = "transactional"
	StreamBroadcast     = "broadcast"
)

// SendEmailRequest is the body of POST /emails.
type SendEmailRequest struct {
	// From is "Name <sender@your-verified-domain.com>". It may come from the template.
	From    string   `json:"from,omitempty"`
	To      []string `json:"to"`
	Cc      []string `json:"cc,omitempty"`
	Bcc     []string `json:"bcc,omitempty"`
	ReplyTo []string `json:"reply_to,omitempty"`
	Subject string   `json:"subject,omitempty"`
	Html    string   `json:"html,omitempty"`
	// Text is generated from Html when omitted.
	Text        string            `json:"text,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	Tags        []Tag             `json:"tags,omitempty"`
	// ScheduledAt is ISO 8601 or natural language ("in 1 hour", "tomorrow at 9am"), within 30 days.
	ScheduledAt string       `json:"scheduled_at,omitempty"`
	Template    *TemplateRef `json:"template,omitempty"`
	// Stream is a Pidgely extension: StreamTransactional (default) or
	// StreamBroadcast (adds RFC 8058 one-click unsubscribe).
	Stream string `json:"stream,omitempty"`
}

// SendEmailResponse is returned by Emails.Send.
type SendEmailResponse struct {
	Id string `json:"id"`
}

// Email is a sent (or scheduled) email.
type Email struct {
	Object    string   `json:"object"`
	Id        string   `json:"id"`
	MessageId string   `json:"message_id"`
	To        []string `json:"to"`
	Cc        []string `json:"cc"`
	Bcc       []string `json:"bcc"`
	ReplyTo   []string `json:"reply_to"`
	From      string   `json:"from"`
	Subject   string   `json:"subject"`
	Html      string   `json:"html"`
	Text      string   `json:"text"`
	// CreatedAt uses the Resend timestamp format ("2026-10-12 16:44:41.000000+00").
	CreatedAt   string `json:"created_at"`
	ScheduledAt string `json:"scheduled_at"`
	// LastEvent is queued, scheduled, sent, delivered, delivery_delayed, bounced,
	// complained, suppressed, failed or canceled.
	LastEvent string `json:"last_event"`
	Tags      []Tag  `json:"tags"`
}

// EmailAttachment describes a stored attachment of a sent email.
type EmailAttachment struct {
	Object             string `json:"object"`
	Id                 string `json:"id"`
	Filename           string `json:"filename"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
	ContentDisposition string `json:"content_disposition"`
	ContentId          string `json:"content_id"`
	// DownloadUrl is a signed link, valid for 1 hour.
	DownloadUrl string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"`
}

// UpdateEmailRequest reschedules a scheduled email.
type UpdateEmailRequest struct {
	ScheduledAt string `json:"scheduled_at"`
}

// EmailsService sends and manages emails.
type EmailsService struct {
	client *Client
	// Attachments reads the stored attachments of sent emails.
	Attachments *EmailAttachmentsService
}

// EmailAttachmentsService reads the attachments of sent emails.
type EmailAttachmentsService struct{ client *Client }

// Send sends an email (POST /emails). Pass [WithIdempotencyKey] to make it safe to retry.
func (s *EmailsService) Send(ctx context.Context, req *SendEmailRequest, opts ...RequestOption) (*SendEmailResponse, error) {
	return send[SendEmailResponse](ctx, s.client, http.MethodPost, "/emails", req, opts)
}

// Get retrieves an email, including its last_event (GET /emails/{id}).
func (s *EmailsService) Get(ctx context.Context, id string, opts ...RequestOption) (*Email, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return get[Email](ctx, s.client, "/emails/"+seg(id), nil, opts)
}

// List lists sent emails, newest first (GET /emails).
func (s *EmailsService) List(ctx context.Context, params *ListParams, opts ...RequestOption) (*List[Email], error) {
	return get[List[Email]](ctx, s.client, "/emails", params.values(), opts)
}

// Update reschedules a scheduled email (PATCH /emails/{id}).
func (s *EmailsService) Update(ctx context.Context, id string, req *UpdateEmailRequest, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPatch, "/emails/"+seg(id), req, opts)
}

// Cancel cancels a scheduled email (POST /emails/{id}/cancel).
func (s *EmailsService) Cancel(ctx context.Context, id string, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPost, "/emails/"+seg(id)+"/cancel", nil, opts)
}

// List lists the attachments of a sent email (GET /emails/{email_id}/attachments).
func (s *EmailAttachmentsService) List(ctx context.Context, emailID string, params *ListParams, opts ...RequestOption) (*List[EmailAttachment], error) {
	if err := required("emailID", emailID); err != nil {
		return nil, err
	}
	return get[List[EmailAttachment]](ctx, s.client, "/emails/"+seg(emailID)+"/attachments", params.values(), opts)
}

// Get retrieves one attachment with a signed DownloadUrl (GET /emails/{email_id}/attachments/{id}).
func (s *EmailAttachmentsService) Get(ctx context.Context, emailID, id string, opts ...RequestOption) (*EmailAttachment, error) {
	if err := required("emailID", emailID); err != nil {
		return nil, err
	}
	if err := required("id", id); err != nil {
		return nil, err
	}
	return get[EmailAttachment](ctx, s.client, "/emails/"+seg(emailID)+"/attachments/"+seg(id), nil, opts)
}

// Batch validation modes for [WithBatchValidation].
const (
	// BatchStrict sends all emails or none (the default).
	BatchStrict = "strict"
	// BatchPermissive sends the valid emails and reports the invalid ones in Errors.
	BatchPermissive = "permissive"
)

// WithBatchValidation sets the x-batch-validation mode of Batch.Send.
func WithBatchValidation(mode string) RequestOption {
	return WithRequestHeader("x-batch-validation", mode)
}

// BatchError reports a rejected batch row: an email in permissive mode, or a
// contact in Contacts.Batch.
type BatchError struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// BatchResponse is returned by Batch.Send.
type BatchResponse struct {
	Data []SendEmailResponse `json:"data"`
	// Errors is set with BatchPermissive.
	Errors []BatchError `json:"errors,omitempty"`
}

// BatchService sends several emails in one request.
type BatchService struct{ client *Client }

// Send sends up to 100 emails (POST /emails/batch).
func (s *BatchService) Send(ctx context.Context, emails []SendEmailRequest, opts ...RequestOption) (*BatchResponse, error) {
	if emails == nil {
		emails = []SendEmailRequest{}
	}
	return send[BatchResponse](ctx, s.client, http.MethodPost, "/emails/batch", emails, opts)
}
