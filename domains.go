package pidgely

import (
	"context"
	"net/http"
)

// Tls modes of a domain.
const (
	TLSOpportunistic = "opportunistic"
	TLSEnforced      = "enforced"
)

// DomainCapabilities lists what a domain is enabled for.
type DomainCapabilities struct {
	Sending   string `json:"sending,omitempty"`
	Receiving string `json:"receiving,omitempty"`
}

// DomainRecord is a DNS record to publish for a domain.
type DomainRecord struct {
	// Record is DKIM, SPF or DMARC.
	Record string `json:"record"`
	// Name is relative to the domain.
	Name string `json:"name"`
	// Type is TXT or MX.
	Type     string `json:"type"`
	Ttl      string `json:"ttl"`
	Status   string `json:"status"`
	Value    string `json:"value"`
	Priority int    `json:"priority,omitempty"`
}

// Domain is a sending domain.
type Domain struct {
	Object string `json:"object"`
	Id     string `json:"id"`
	Name   string `json:"name"`
	// Status is not_started, pending, verified or failed.
	Status        string             `json:"status"`
	CreatedAt     string             `json:"created_at"`
	Region        string             `json:"region"`
	OpenTracking  bool               `json:"open_tracking"`
	ClickTracking bool               `json:"click_tracking"`
	Tls           string             `json:"tls"`
	Capabilities  DomainCapabilities `json:"capabilities"`
	// Records are the DNS records to publish. Present on create/get, omitted in lists.
	Records []DomainRecord `json:"records,omitempty"`
}

// CreateDomainRequest is the body of POST /domains.
type CreateDomainRequest struct {
	Name   string `json:"name"`
	Region string `json:"region,omitempty"`
	// CustomReturnPath is the bounce subdomain label (default "bounces").
	CustomReturnPath string `json:"custom_return_path,omitempty"`
	// Tls is TLSOpportunistic (default) or TLSEnforced.
	Tls string `json:"tls,omitempty"`
	// OpenTracking is accepted but not supported yet.
	OpenTracking bool `json:"open_tracking,omitempty"`
	// ClickTracking is accepted but not supported yet.
	ClickTracking bool                `json:"click_tracking,omitempty"`
	Capabilities  *DomainCapabilities `json:"capabilities,omitempty"`
}

// UpdateDomainRequest is the body of PATCH /domains/{id}.
type UpdateDomainRequest struct {
	Tls           string `json:"tls,omitempty"`
	OpenTracking  bool   `json:"open_tracking,omitempty"`
	ClickTracking bool   `json:"click_tracking,omitempty"`
}

// DomainsService manages sending domains.
type DomainsService struct{ client *Client }

// Create adds a domain and returns the DNS records to publish (POST /domains).
func (s *DomainsService) Create(ctx context.Context, req *CreateDomainRequest, opts ...RequestOption) (*Domain, error) {
	return send[Domain](ctx, s.client, http.MethodPost, "/domains", req, opts)
}

// Get retrieves a domain with its DNS records and their status (GET /domains/{id}).
func (s *DomainsService) Get(ctx context.Context, id string, opts ...RequestOption) (*Domain, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return get[Domain](ctx, s.client, "/domains/"+seg(id), nil, opts)
}

// List lists domains (GET /domains).
func (s *DomainsService) List(ctx context.Context, params *ListParams, opts ...RequestOption) (*List[Domain], error) {
	return get[List[Domain]](ctx, s.client, "/domains", params.values(), opts)
}

// Update changes a domain's settings (PATCH /domains/{id}).
func (s *DomainsService) Update(ctx context.Context, id string, req *UpdateDomainRequest, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPatch, "/domains/"+seg(id), req, opts)
}

// Remove deletes a domain (DELETE /domains/{id}).
func (s *DomainsService) Remove(ctx context.Context, id string, opts ...RequestOption) (*Deleted, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[Deleted](ctx, s.client, http.MethodDelete, "/domains/"+seg(id), nil, opts)
}

// Verify checks the domain's DNS records now (POST /domains/{id}/verify).
func (s *DomainsService) Verify(ctx context.Context, id string, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPost, "/domains/"+seg(id)+"/verify", nil, opts)
}
