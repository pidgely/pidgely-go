package pidgely

import (
	"context"
	"net/http"
	"net/url"
)

// Suppression origins.
const (
	OriginBounce      = "bounce"
	OriginComplaint   = "complaint"
	OriginManual      = "manual"
	OriginUnsubscribe = "unsubscribe"
)

// Suppression is an address Pidgely will not send to.
type Suppression struct {
	Object string `json:"object"`
	Id     string `json:"id"`
	Email  string `json:"email"`
	// Origin is bounce, complaint, manual or unsubscribe.
	Origin    string `json:"origin"`
	SourceId  string `json:"source_id"`
	CreatedAt string `json:"created_at"`
}

// AddSuppressionRequest is the body of POST /suppressions.
type AddSuppressionRequest struct {
	Email string `json:"email"`
}

// ListSuppressionsParams filters GET /suppressions.
type ListSuppressionsParams struct {
	ListParams
	// Origin filters by OriginBounce, OriginComplaint, OriginManual or OriginUnsubscribe.
	Origin string
}

// AddSuppressionBatchRequest adds up to 100 addresses.
type AddSuppressionBatchRequest struct {
	Emails []string `json:"emails"`
}

// RemoveSuppressionBatchRequest removes up to 100 entries by email and/or id.
type RemoveSuppressionBatchRequest struct {
	Emails []string `json:"emails,omitempty"`
	Ids    []string `json:"ids,omitempty"`
}

// SuppressionsService manages the suppression list.
type SuppressionsService struct {
	client *Client
	// Batch adds and removes many entries at once.
	Batch *SuppressionsBatchService
}

// Add stops sending to an address (POST /suppressions).
func (s *SuppressionsService) Add(ctx context.Context, req *AddSuppressionRequest, opts ...RequestOption) (*ObjectRef, error) {
	return send[ObjectRef](ctx, s.client, http.MethodPost, "/suppressions", req, opts)
}

// List lists suppressions, optionally filtered by origin (GET /suppressions).
func (s *SuppressionsService) List(ctx context.Context, params *ListSuppressionsParams, opts ...RequestOption) (*List[Suppression], error) {
	var q url.Values
	if params == nil {
		q = (*ListParams)(nil).values()
	} else {
		q = params.values()
		if params.Origin != "" {
			q.Set("origin", params.Origin)
		}
	}
	return get[List[Suppression]](ctx, s.client, "/suppressions", q, opts)
}

// Get retrieves a suppression by id or email (GET /suppressions/{id_or_email}).
func (s *SuppressionsService) Get(ctx context.Context, idOrEmail string, opts ...RequestOption) (*Suppression, error) {
	if err := required("idOrEmail", idOrEmail); err != nil {
		return nil, err
	}
	return get[Suppression](ctx, s.client, "/suppressions/"+seg(idOrEmail), nil, opts)
}

// Remove allows sending to an address again (DELETE /suppressions/{id_or_email}).
func (s *SuppressionsService) Remove(ctx context.Context, idOrEmail string, opts ...RequestOption) (*Deleted, error) {
	if err := required("idOrEmail", idOrEmail); err != nil {
		return nil, err
	}
	return send[Deleted](ctx, s.client, http.MethodDelete, "/suppressions/"+seg(idOrEmail), nil, opts)
}

// SuppressionsBatchService adds and removes suppressions in bulk.
type SuppressionsBatchService struct{ client *Client }

// Add adds up to 100 addresses (POST /suppressions/batch/add).
func (s *SuppressionsBatchService) Add(ctx context.Context, req *AddSuppressionBatchRequest, opts ...RequestOption) (*Data[ObjectRef], error) {
	return send[Data[ObjectRef]](ctx, s.client, http.MethodPost, "/suppressions/batch/add", req, opts)
}

// Remove removes up to 100 entries by email and/or id (POST /suppressions/batch/remove).
func (s *SuppressionsBatchService) Remove(ctx context.Context, req *RemoveSuppressionBatchRequest, opts ...RequestOption) (*Data[Deleted], error) {
	return send[Data[Deleted]](ctx, s.client, http.MethodPost, "/suppressions/batch/remove", req, opts)
}
