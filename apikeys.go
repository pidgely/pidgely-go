package pidgely

import (
	"context"
	"net/http"
)

// API key permissions.
const (
	PermissionFullAccess    = "full_access"
	PermissionSendingAccess = "sending_access"
)

// ApiKey is an API key (without its secret token).
type ApiKey struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Permission string `json:"permission"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
}

// CreatedApiKey is returned once, when a key is created.
type CreatedApiKey struct {
	Object string `json:"object"`
	Id     string `json:"id"`
	// Token is the secret pdg_… key. It is only shown once.
	Token string `json:"token"`
}

// CreateApiKeyRequest is the body of POST /api-keys.
type CreateApiKeyRequest struct {
	Name string `json:"name"`
	// Permission is PermissionFullAccess (default) or PermissionSendingAccess.
	Permission string `json:"permission,omitempty"`
	// DomainId restricts a sending_access key to one domain.
	DomainId string `json:"domain_id,omitempty"`
}

// UpdateApiKeyRequest renames a key.
type UpdateApiKeyRequest struct {
	Name string `json:"name"`
}

// ApiKeysService manages API keys.
type ApiKeysService struct{ client *Client }

// Create creates an API key; the token is only returned once (POST /api-keys).
func (s *ApiKeysService) Create(ctx context.Context, req *CreateApiKeyRequest, opts ...RequestOption) (*CreatedApiKey, error) {
	return send[CreatedApiKey](ctx, s.client, http.MethodPost, "/api-keys", req, opts)
}

// List lists API keys (GET /api-keys).
func (s *ApiKeysService) List(ctx context.Context, params *ListParams, opts ...RequestOption) (*List[ApiKey], error) {
	return get[List[ApiKey]](ctx, s.client, "/api-keys", params.values(), opts)
}

// Update renames an API key (PATCH /api-keys/{id}).
func (s *ApiKeysService) Update(ctx context.Context, id string, req *UpdateApiKeyRequest, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPatch, "/api-keys/"+seg(id), req, opts)
}

// Remove revokes an API key (DELETE /api-keys/{id}).
func (s *ApiKeysService) Remove(ctx context.Context, id string, opts ...RequestOption) (*Deleted, error) {
	if err := required("id", id); err != nil {
		return nil, err
	}
	return send[Deleted](ctx, s.client, http.MethodDelete, "/api-keys/"+seg(id), nil, opts)
}
