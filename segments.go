package pidgely

import (
	"context"
	"net/http"
)

// Segment is a named group of contacts.
//
// As an input (CreateContactRequest.Segments) only Id is needed.
type Segment struct {
	Object    string `json:"object,omitempty"`
	Id        string `json:"id"`
	Name      string `json:"name,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Audience is the legacy name of Segment.
//
// Deprecated: use Segment.
type Audience = Segment

// CreateAudienceRequest is the legacy name of CreateSegmentRequest.
//
// Deprecated: use CreateSegmentRequest.
type CreateAudienceRequest = CreateSegmentRequest

// CreateSegmentRequest is the body of POST /segments.
type CreateSegmentRequest struct {
	Name string `json:"name"`
}

// UpdateSegmentRequest renames a segment.
type UpdateSegmentRequest struct {
	Name string `json:"name"`
}

// CreatedSegment is returned by Segments.Create.
type CreatedSegment struct {
	Object string `json:"object"`
	Id     string `json:"id"`
	Name   string `json:"name"`
}

// SegmentsService manages segments (also exposed as the legacy Audiences).
type SegmentsService struct{ client *Client }

// Create creates a segment (POST /segments).
func (s *SegmentsService) Create(ctx context.Context, req *CreateSegmentRequest, opts ...RequestOption) (*CreatedSegment, error) {
	return send[CreatedSegment](ctx, s.client, http.MethodPost, "/segments", req, opts)
}

// List lists segments (GET /segments).
func (s *SegmentsService) List(ctx context.Context, params *ListParams, opts ...RequestOption) (*List[Segment], error) {
	return get[List[Segment]](ctx, s.client, "/segments", params.values(), opts)
}

// Get retrieves a segment (GET /segments/{id}).
func (s *SegmentsService) Get(ctx context.Context, id string, opts ...RequestOption) (*Segment, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return get[Segment](ctx, s.client, "/segments/"+seg(id), nil, opts)
}

// Update renames a segment (PATCH /segments/{id}).
func (s *SegmentsService) Update(ctx context.Context, id string, req *UpdateSegmentRequest, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPatch, "/segments/"+seg(id), req, opts)
}

// Remove deletes a segment (DELETE /segments/{id}).
func (s *SegmentsService) Remove(ctx context.Context, id string, opts ...RequestOption) (*Deleted, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[Deleted](ctx, s.client, http.MethodDelete, "/segments/"+seg(id), nil, opts)
}

// ListContacts lists the contacts of a segment (GET /segments/{id}/contacts).
func (s *SegmentsService) ListContacts(ctx context.Context, id string, params *ListParams, opts ...RequestOption) (*List[Contact], error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return get[List[Contact]](ctx, s.client, "/segments/"+seg(id)+"/contacts", params.values(), opts)
}
