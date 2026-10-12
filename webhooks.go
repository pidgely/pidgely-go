package pidgely

import (
	"context"
	"net/http"

	"github.com/pidgely/pidgely-go/webhook"
)

// Webhook event types.
const (
	EventEmailSent            = "email.sent"
	EventEmailDelivered       = "email.delivered"
	EventEmailDeliveryDelayed = "email.delivery_delayed"
	EventEmailBounced         = "email.bounced"
	EventEmailComplained      = "email.complained"
	EventEmailFailed          = "email.failed"
	EventEmailSuppressed      = "email.suppressed"
	EventDomainUpdated        = "domain.updated"
	EventContactCreated       = "contact.created"
	EventContactUpdated       = "contact.updated"
	EventContactDeleted       = "contact.deleted"
)

// Webhook is a webhook endpoint.
type Webhook struct {
	Object    string `json:"object"`
	Id        string `json:"id"`
	CreatedAt string `json:"created_at"`
	// Status is enabled or disabled.
	Status        string   `json:"status"`
	Endpoint      string   `json:"endpoint"`
	Events        []string `json:"events"`
	SigningSecret string   `json:"signing_secret,omitempty"`
}

// WebhookSecret is returned when a webhook is created or its secret rotated.
type WebhookSecret struct {
	Object string `json:"object"`
	Id     string `json:"id"`
	// SigningSecret is the whsec_… secret used to verify deliveries.
	SigningSecret string `json:"signing_secret"`
}

// CreateWebhookRequest is the body of POST /webhooks.
type CreateWebhookRequest struct {
	Endpoint string   `json:"endpoint"`
	Events   []string `json:"events"`
}

// UpdateWebhookRequest is the body of PATCH /webhooks/{id}. Empty fields are left unchanged.
type UpdateWebhookRequest struct {
	Endpoint string   `json:"endpoint,omitempty"`
	Events   []string `json:"events,omitempty"`
	// Status is "enabled" or "disabled".
	Status string `json:"status,omitempty"`
}

// WebhooksService manages webhook endpoints.
type WebhooksService struct{ client *Client }

// Create registers an endpoint and returns its signing secret (POST /webhooks).
func (s *WebhooksService) Create(ctx context.Context, req *CreateWebhookRequest, opts ...RequestOption) (*WebhookSecret, error) {
	return send[WebhookSecret](ctx, s.client, http.MethodPost, "/webhooks", req, opts)
}

// List lists webhook endpoints (GET /webhooks).
func (s *WebhooksService) List(ctx context.Context, params *ListParams, opts ...RequestOption) (*List[Webhook], error) {
	return get[List[Webhook]](ctx, s.client, "/webhooks", params.values(), opts)
}

// Get retrieves a webhook endpoint (GET /webhooks/{id}).
func (s *WebhooksService) Get(ctx context.Context, id string, opts ...RequestOption) (*Webhook, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return get[Webhook](ctx, s.client, "/webhooks/"+seg(id), nil, opts)
}

// Update changes a webhook endpoint (PATCH /webhooks/{id}).
func (s *WebhooksService) Update(ctx context.Context, id string, req *UpdateWebhookRequest, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPatch, "/webhooks/"+seg(id), req, opts)
}

// Remove deletes a webhook endpoint (DELETE /webhooks/{id}).
func (s *WebhooksService) Remove(ctx context.Context, id string, opts ...RequestOption) (*Deleted, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[Deleted](ctx, s.client, http.MethodDelete, "/webhooks/"+seg(id), nil, opts)
}

// RotateSigningSecret issues a new signing secret (POST /webhooks/{id}/signing-secret/rotate).
func (s *WebhooksService) RotateSigningSecret(ctx context.Context, id string, opts ...RequestOption) (*WebhookSecret, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[WebhookSecret](ctx, s.client, http.MethodPost, "/webhooks/"+seg(id)+"/signing-secret/rotate", nil, opts)
}

// Verify checks a delivery's signature. It is a shortcut for [webhook.Verify].
func (s *WebhooksService) Verify(payload []byte, headers http.Header, secret string, opts ...webhook.Option) error {
	return webhook.Verify(payload, headers, secret, opts...)
}
