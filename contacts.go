package pidgely

import (
	"context"
	"net/http"
)

// ContactProperty is a typed custom property value.
type ContactProperty struct {
	Value any `json:"value"`
	// Type is string, number or boolean.
	Type string `json:"type"`
}

// Contact is a marketing contact.
type Contact struct {
	Object       string                     `json:"object"`
	Id           string                     `json:"id"`
	Email        string                     `json:"email"`
	FirstName    string                     `json:"first_name"`
	LastName     string                     `json:"last_name"`
	CreatedAt    string                     `json:"created_at"`
	Unsubscribed bool                       `json:"unsubscribed"`
	Properties   map[string]ContactProperty `json:"properties,omitempty"`
}

// CreateContactRequest is the body of POST /contacts. It creates the contact or
// updates the existing one with the same email.
type CreateContactRequest struct {
	Email        string `json:"email"`
	FirstName    string `json:"first_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	Unsubscribed *bool  `json:"unsubscribed,omitempty"`
	// Properties are custom values: string, number, bool or nil.
	Properties map[string]any `json:"properties,omitempty"`
	// Segments to add the contact to.
	Segments []Segment `json:"segments,omitempty"`
	// AudienceId creates the contact through the legacy /audiences/{id}/contacts route.
	//
	// Deprecated: use Segments.
	AudienceId string `json:"-"`
}

// UpdateContactRequest is the body of PATCH /contacts/{id_or_email}. Empty and nil fields are left unchanged.
type UpdateContactRequest struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	// Unsubscribed is kept in sync with the suppression list. Use pidgely.Bool(true).
	Unsubscribed *bool          `json:"unsubscribed,omitempty"`
	Properties   map[string]any `json:"properties,omitempty"`
	// AudienceId updates the contact through the legacy /audiences/{id}/contacts route.
	//
	// Deprecated: contacts are global; omit it.
	AudienceId string `json:"-"`
}

// ContactInput is one row of ContactBatchRequest. It creates the contact or
// updates the existing one with the same email.
type ContactInput struct {
	Email        string `json:"email"`
	FirstName    string `json:"first_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	Unsubscribed *bool  `json:"unsubscribed,omitempty"`
	// Properties are custom values: string, number, bool or nil.
	Properties map[string]any `json:"properties,omitempty"`
	// Segments to add the contact to.
	Segments []Segment `json:"segments,omitempty"`
}

// ContactBatchRequest is the body of POST /contacts/batch.
type ContactBatchRequest struct {
	// Contacts are up to 1,000 rows, each saved independently.
	Contacts []ContactInput `json:"contacts"`
	// SegmentId adds every row to this segment.
	SegmentId string `json:"segment_id,omitempty"`
}

// ContactBatchItem is a saved row of a contact batch.
type ContactBatchItem struct {
	// Index is the position of the row in ContactBatchRequest.Contacts.
	Index int    `json:"index"`
	Id    string `json:"id"`
}

// ContactBatchResult is returned by Contacts.Batch. Invalid rows are reported
// in Errors with their index; the other rows are saved.
type ContactBatchResult struct {
	Object string             `json:"object"`
	Data   []ContactBatchItem `json:"data"`
	Errors []BatchError       `json:"errors"`
}

// ListContactsParams filters GET /contacts.
type ListContactsParams struct {
	ListParams
	// SegmentId lists only the contacts of a segment.
	SegmentId string
	// AudienceId is the legacy name of SegmentId.
	//
	// Deprecated: use SegmentId.
	AudienceId string
}

// ContactDeleted is returned when a contact is removed.
type ContactDeleted struct {
	Object  string `json:"object"`
	Contact string `json:"contact"`
	Id      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// ContactSegmentRemoved is returned when a contact leaves a segment.
type ContactSegmentRemoved struct {
	Object    string `json:"object"`
	Id        string `json:"id"`
	SegmentId string `json:"segment_id"`
	Deleted   bool   `json:"deleted"`
}

// ContactsService manages contacts. Contacts are addressed by id or by email.
type ContactsService struct {
	client *Client
	// Segments manages the segment memberships of a contact.
	Segments *ContactSegmentsService
}

func contactsBase(audienceID string) string {
	if audienceID != "" {
		return "/audiences/" + seg(audienceID) + "/contacts"
	}
	return "/contacts"
}

// Create creates or updates a contact by email (POST /contacts).
func (s *ContactsService) Create(ctx context.Context, req *CreateContactRequest, opts ...RequestOption) (*ObjectRef, error) {
	path := "/contacts"
	if req != nil {
		path = contactsBase(req.AudienceId)
	}
	return send[ObjectRef](ctx, s.client, http.MethodPost, path, req, opts)
}

// Batch creates or updates up to 1,000 contacts by email (POST /contacts/batch).
// Each row is saved independently; invalid rows are reported in the result's
// Errors with their index instead of failing the request.
func (s *ContactsService) Batch(ctx context.Context, req *ContactBatchRequest, opts ...RequestOption) (*ContactBatchResult, error) {
	if req == nil {
		req = &ContactBatchRequest{}
	}
	if req.Contacts == nil {
		r := *req
		r.Contacts = []ContactInput{}
		req = &r
	}
	return send[ContactBatchResult](ctx, s.client, http.MethodPost, "/contacts/batch", req, opts)
}

// List lists contacts, optionally filtered by segment (GET /contacts).
func (s *ContactsService) List(ctx context.Context, params *ListContactsParams, opts ...RequestOption) (*List[Contact], error) {
	if params == nil {
		params = &ListContactsParams{}
	}
	q := params.values()
	if params.SegmentId != "" {
		q.Set("segment_id", params.SegmentId)
	}
	if params.AudienceId != "" {
		q.Set("audience_id", params.AudienceId)
	}
	return get[List[Contact]](ctx, s.client, "/contacts", q, opts)
}

// Get retrieves a contact by id or email (GET /contacts/{id_or_email}).
func (s *ContactsService) Get(ctx context.Context, idOrEmail string, opts ...RequestOption) (*Contact, error) {
	if err := required("idOrEmail", idOrEmail); err != nil {
		return nil, err
	}
	return get[Contact](ctx, s.client, "/contacts/"+seg(idOrEmail), nil, opts)
}

// Update changes a contact by id or email (PATCH /contacts/{id_or_email}).
func (s *ContactsService) Update(ctx context.Context, idOrEmail string, req *UpdateContactRequest, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("idOrEmail", idOrEmail); err != nil {
		return nil, err
	}
	base := "/contacts"
	if req != nil {
		base = contactsBase(req.AudienceId)
	}
	return send[ObjectRef](ctx, s.client, http.MethodPatch, base+"/"+seg(idOrEmail), req, opts)
}

// Remove deletes a contact by id or email (DELETE /contacts/{id_or_email}).
func (s *ContactsService) Remove(ctx context.Context, idOrEmail string, opts ...RequestOption) (*ContactDeleted, error) {
	if err := required("idOrEmail", idOrEmail); err != nil {
		return nil, err
	}
	return send[ContactDeleted](ctx, s.client, http.MethodDelete, "/contacts/"+seg(idOrEmail), nil, opts)
}

// ContactSegmentsService manages the segments of a contact.
type ContactSegmentsService struct{ client *Client }

// List lists the segments a contact belongs to (GET /contacts/{id_or_email}/segments).
func (s *ContactSegmentsService) List(ctx context.Context, idOrEmail string, params *ListParams, opts ...RequestOption) (*List[Segment], error) {
	if err := required("idOrEmail", idOrEmail); err != nil {
		return nil, err
	}
	return get[List[Segment]](ctx, s.client, "/contacts/"+seg(idOrEmail)+"/segments", params.values(), opts)
}

// Add adds a contact to a segment (POST /contacts/{id_or_email}/segments/{segment_id}).
func (s *ContactSegmentsService) Add(ctx context.Context, idOrEmail, segmentID string, opts ...RequestOption) (*IDResponse, error) {
	if err := required("idOrEmail", idOrEmail); err != nil {
		return nil, err
	}
	if err := required("segmentID", segmentID); err != nil {
		return nil, err
	}
	return send[IDResponse](ctx, s.client, http.MethodPost, "/contacts/"+seg(idOrEmail)+"/segments/"+seg(segmentID), nil, opts)
}

// Remove removes a contact from a segment (DELETE /contacts/{id_or_email}/segments/{segment_id}).
func (s *ContactSegmentsService) Remove(ctx context.Context, idOrEmail, segmentID string, opts ...RequestOption) (*ContactSegmentRemoved, error) {
	if err := required("idOrEmail", idOrEmail); err != nil {
		return nil, err
	}
	if err := required("segmentID", segmentID); err != nil {
		return nil, err
	}
	return send[ContactSegmentRemoved](ctx, s.client, http.MethodDelete, "/contacts/"+seg(idOrEmail)+"/segments/"+seg(segmentID), nil, opts)
}
