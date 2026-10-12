package pidgely

import (
	"context"
	"net/http"
)

// TemplateVariable is a declared template variable.
type TemplateVariable struct {
	Id  string `json:"id"`
	Key string `json:"key"`
	// Type is string or number.
	Type          string `json:"type"`
	FallbackValue any    `json:"fallback_value"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// TemplateVariableInput declares a template variable.
type TemplateVariableInput struct {
	Key string `json:"key"`
	// Type is "string" or "number".
	Type string `json:"type"`
	// FallbackValue is used when the variable is not given (string or number).
	FallbackValue any `json:"fallback_value,omitempty"`
}

// Template is a reusable email template.
type Template struct {
	Object           string `json:"object"`
	Id               string `json:"id"`
	CurrentVersionId string `json:"current_version_id"`
	Alias            string `json:"alias"`
	Name             string `json:"name"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	// Status is draft or published.
	Status                 string             `json:"status"`
	PublishedAt            string             `json:"published_at"`
	From                   string             `json:"from"`
	Subject                string             `json:"subject"`
	ReplyTo                []string           `json:"reply_to"`
	Html                   string             `json:"html"`
	Text                   string             `json:"text"`
	Variables              []TemplateVariable `json:"variables"`
	HasUnpublishedVersions bool               `json:"has_unpublished_versions"`
}

// CreateTemplateRequest is the body of POST /templates.
type CreateTemplateRequest struct {
	Name      string                  `json:"name"`
	Alias     string                  `json:"alias,omitempty"`
	From      string                  `json:"from,omitempty"`
	Subject   string                  `json:"subject,omitempty"`
	ReplyTo   []string                `json:"reply_to,omitempty"`
	Html      string                  `json:"html,omitempty"`
	Text      string                  `json:"text,omitempty"`
	Variables []TemplateVariableInput `json:"variables,omitempty"`
}

// UpdateTemplateRequest is the body of PATCH /templates/{id_or_alias}. Empty fields are left unchanged.
type UpdateTemplateRequest struct {
	Name      string                  `json:"name,omitempty"`
	Alias     string                  `json:"alias,omitempty"`
	From      string                  `json:"from,omitempty"`
	Subject   string                  `json:"subject,omitempty"`
	ReplyTo   []string                `json:"reply_to,omitempty"`
	Html      string                  `json:"html,omitempty"`
	Text      string                  `json:"text,omitempty"`
	Variables []TemplateVariableInput `json:"variables,omitempty"`
}

// TemplatesService manages templates. Templates are addressed by id or alias.
type TemplatesService struct{ client *Client }

// Create creates a draft template (POST /templates). Publish it with Publish.
func (s *TemplatesService) Create(ctx context.Context, req *CreateTemplateRequest, opts ...RequestOption) (*ObjectRef, error) {
	return send[ObjectRef](ctx, s.client, http.MethodPost, "/templates", req, opts)
}

// List lists templates (GET /templates).
func (s *TemplatesService) List(ctx context.Context, params *ListParams, opts ...RequestOption) (*List[Template], error) {
	return get[List[Template]](ctx, s.client, "/templates", params.values(), opts)
}

// Get retrieves a template's current draft by id or alias (GET /templates/{id_or_alias}).
func (s *TemplatesService) Get(ctx context.Context, idOrAlias string, opts ...RequestOption) (*Template, error) {
	if err := required("idOrAlias", idOrAlias); err != nil {
		return nil, err
	}
	return get[Template](ctx, s.client, "/templates/"+seg(idOrAlias), nil, opts)
}

// Update changes a template's draft (PATCH /templates/{id_or_alias}).
func (s *TemplatesService) Update(ctx context.Context, idOrAlias string, req *UpdateTemplateRequest, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("idOrAlias", idOrAlias); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPatch, "/templates/"+seg(idOrAlias), req, opts)
}

// Remove deletes a template (DELETE /templates/{id_or_alias}).
func (s *TemplatesService) Remove(ctx context.Context, idOrAlias string, opts ...RequestOption) (*Deleted, error) {
	if err := required("idOrAlias", idOrAlias); err != nil {
		return nil, err
	}
	return send[Deleted](ctx, s.client, http.MethodDelete, "/templates/"+seg(idOrAlias), nil, opts)
}

// Publish publishes the draft; the published version is what TemplateRef sends
// (POST /templates/{id_or_alias}/publish).
func (s *TemplatesService) Publish(ctx context.Context, idOrAlias string, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("idOrAlias", idOrAlias); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPost, "/templates/"+seg(idOrAlias)+"/publish", nil, opts)
}

// Duplicate copies a template as a new draft (POST /templates/{id_or_alias}/duplicate).
func (s *TemplatesService) Duplicate(ctx context.Context, idOrAlias string, opts ...RequestOption) (*ObjectRef, error) {
	if err := required("idOrAlias", idOrAlias); err != nil {
		return nil, err
	}
	return send[ObjectRef](ctx, s.client, http.MethodPost, "/templates/"+seg(idOrAlias)+"/duplicate", nil, opts)
}
