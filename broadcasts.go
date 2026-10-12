package pidgely

import (
	"context"
	"net/http"
)

// Broadcast is a marketing email sent to a segment.
type Broadcast struct {
	Object      string   `json:"object"`
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	SegmentId   string   `json:"segment_id"`
	AudienceId  string   `json:"audience_id"`
	From        string   `json:"from"`
	Subject     string   `json:"subject"`
	ReplyTo     []string `json:"reply_to"`
	PreviewText string   `json:"preview_text"`
	Html        string   `json:"html"`
	Text        string   `json:"text"`
	// Status is draft, scheduled, queued, sending, sent, canceled or failed.
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	ScheduledAt string `json:"scheduled_at"`
	SentAt      string `json:"sent_at"`
	Recipients  int    `json:"recipients"`
	// Stats counts deliveries by status (Pidgely extension).
	Stats map[string]int `json:"stats,omitempty"`
}

// CreateBroadcastRequest is the body of POST /broadcasts.
type CreateBroadcastRequest struct {
	SegmentId string `json:"segment_id,omitempty"`
	// AudienceId is the legacy alias of SegmentId.
	//
	// Deprecated: use SegmentId.
	AudienceId string   `json:"audience_id,omitempty"`
	From       string   `json:"from"`
	Subject    string   `json:"subject"`
	ReplyTo    []string `json:"reply_to,omitempty"`
	// Html supports {{{FIRST_NAME|there}}}, {{{LAST_NAME}}}, {{{EMAIL}}},
	// {{{contact.<property>}}} and {{{PIDGELY_UNSUBSCRIBE_URL}}}.
	Html        string `json:"html,omitempty"`
	Text        string `json:"text,omitempty"`
	Name        string `json:"name,omitempty"`
	PreviewText string `json:"preview_text,omitempty"`
	// Send sends right away (or at ScheduledAt) instead of creating a draft.
	Send        bool   `json:"send,omitempty"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
}

// UpdateBroadcastRequest is the body of PATCH /broadcasts/{id} (drafts only). Empty fields are left unchanged.
type UpdateBroadcastRequest struct {
	SegmentId   string   `json:"segment_id,omitempty"`
	AudienceId  string   `json:"audience_id,omitempty"`
	From        string   `json:"from,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	ReplyTo     []string `json:"reply_to,omitempty"`
	Html        string   `json:"html,omitempty"`
	Text        string   `json:"text,omitempty"`
	Name        string   `json:"name,omitempty"`
	PreviewText string   `json:"preview_text,omitempty"`
}

// SendBroadcastRequest is the body of POST /broadcasts/{id}/send.
type SendBroadcastRequest struct {
	// ScheduledAt is ISO 8601 or natural language. Empty sends now.
	ScheduledAt string `json:"scheduled_at,omitempty"`
}

// BroadcastsService creates and sends broadcasts.
type BroadcastsService struct{ client *Client }

// Create creates a draft, or sends it with Send: true (POST /broadcasts).
func (s *BroadcastsService) Create(ctx context.Context, req *CreateBroadcastRequest, opts ...RequestOption) (*ObjectRef, error) {
	return send[ObjectRef](ctx, s.client, http.MethodPost, "/broadcasts", req, opts)
}

// List lists broadcasts (GET /broadcasts).
func (s *BroadcastsService) List(ctx context.Context, params *ListParams, opts ...RequestOption) (*List[Broadcast], error) {
	return get[List[Broadcast]](ctx, s.client, "/broadcasts", params.values(), opts)
}

// Get retrieves a broadcast with its per-status stats (GET /broadcasts/{id}).
func (s *BroadcastsService) Get(ctx context.Context, id string, opts ...RequestOption) (*Broadcast, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return get[Broadcast](ctx, s.client, "/broadcasts/"+seg(id), nil, opts)
}

// Update changes a draft broadcast (PATCH /broadcasts/{id}).
func (s *BroadcastsService) Update(ctx context.Context, id string, req *UpdateBroadcastRequest, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPatch, "/broadcasts/"+seg(id), req, opts)
}

// Remove deletes a broadcast (DELETE /broadcasts/{id}).
func (s *BroadcastsService) Remove(ctx context.Context, id string, opts ...RequestOption) (*Deleted, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[Deleted](ctx, s.client, http.MethodDelete, "/broadcasts/"+seg(id), nil, opts)
}

// Send sends a broadcast now, or at req.ScheduledAt (POST /broadcasts/{id}/send). req may be nil.
func (s *BroadcastsService) Send(ctx context.Context, id string, req *SendBroadcastRequest, opts ...RequestOption) (*IDResponse, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	if req == nil {
		req = &SendBroadcastRequest{}
	}
	return send[IDResponse](ctx, s.client, http.MethodPost, "/broadcasts/"+seg(id)+"/send", req, opts)
}

// Cancel cancels a scheduled or queued broadcast (POST /broadcasts/{id}/cancel).
func (s *BroadcastsService) Cancel(ctx context.Context, id string, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPost, "/broadcasts/"+seg(id)+"/cancel", nil, opts)
}

// Duplicate copies a broadcast as a new draft (POST /broadcasts/{id}/duplicate).
func (s *BroadcastsService) Duplicate(ctx context.Context, id string, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPost, "/broadcasts/"+seg(id)+"/duplicate", nil, opts)
}
