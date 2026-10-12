package pidgely

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pidgely/pidgely-go/webhook"
)

// Every endpoint: method, escaped path, query and JSON body.
func TestEndpoints(t *testing.T) {
	ctx := context.Background()
	page := &ListParams{Limit: 10, After: "cur"}
	cases := []struct {
		name   string
		call   func(c *Client) error
		method string
		path   string
		query  string
		body   string // "" means no body
		header [2]string
	}{
		// Emails
		{"emails.get", func(c *Client) error { _, err := c.Emails.Get(ctx, "e 1"); return err }, "GET", "/emails/e%201", "", "", [2]string{}},
		{"emails.list", func(c *Client) error { _, err := c.Emails.List(ctx, page); return err }, "GET", "/emails", "after=cur&limit=10", "", [2]string{}},
		{"emails.list nil", func(c *Client) error { _, err := c.Emails.List(ctx, nil); return err }, "GET", "/emails", "", "", [2]string{}},
		{"emails.update", func(c *Client) error {
			_, err := c.Emails.Update(ctx, "e1", &UpdateEmailRequest{ScheduledAt: "in 2 hours"})
			return err
		}, "PATCH", "/emails/e1", "", `{"scheduled_at":"in 2 hours"}`, [2]string{}},
		{"emails.cancel", func(c *Client) error { _, err := c.Emails.Cancel(ctx, "e1"); return err }, "POST", "/emails/e1/cancel", "", "", [2]string{}},
		{"emails.attachments.list", func(c *Client) error { _, err := c.Emails.Attachments.List(ctx, "e1", nil); return err }, "GET", "/emails/e1/attachments", "", "", [2]string{}},
		{"emails.attachments.get", func(c *Client) error { _, err := c.Emails.Attachments.Get(ctx, "e1", "a1"); return err }, "GET", "/emails/e1/attachments/a1", "", "", [2]string{}},
		{"batch.send", func(c *Client) error {
			_, err := c.Batch.Send(ctx, []SendEmailRequest{{From: "a@acme.com", To: []string{"b@x.y"}, Subject: "s", Html: "h"}}, WithBatchValidation(BatchPermissive))
			return err
		}, "POST", "/emails/batch", "", `[{"from":"a@acme.com","to":["b@x.y"],"subject":"s","html":"h"}]`, [2]string{"X-Batch-Validation", "permissive"}},
		{"batch.send nil", func(c *Client) error { _, err := c.Batch.Send(ctx, nil); return err }, "POST", "/emails/batch", "", `[]`, [2]string{}},
		// Domains
		{"domains.create", func(c *Client) error {
			_, err := c.Domains.Create(ctx, &CreateDomainRequest{Name: "acme.com", CustomReturnPath: "bounces", Tls: TLSOpportunistic, OpenTracking: false})
			return err
		}, "POST", "/domains", "", `{"name":"acme.com","custom_return_path":"bounces","tls":"opportunistic"}`, [2]string{}},
		{"domains.get", func(c *Client) error { _, err := c.Domains.Get(ctx, "d1"); return err }, "GET", "/domains/d1", "", "", [2]string{}},
		{"domains.list", func(c *Client) error { _, err := c.Domains.List(ctx, &ListParams{Before: "x"}); return err }, "GET", "/domains", "before=x", "", [2]string{}},
		{"domains.update", func(c *Client) error {
			_, err := c.Domains.Update(ctx, "d1", &UpdateDomainRequest{Tls: TLSEnforced})
			return err
		}, "PATCH", "/domains/d1", "", `{"tls":"enforced"}`, [2]string{}},
		{"domains.remove", func(c *Client) error { _, err := c.Domains.Remove(ctx, "d1"); return err }, "DELETE", "/domains/d1", "", "", [2]string{}},
		{"domains.verify", func(c *Client) error { _, err := c.Domains.Verify(ctx, "d1"); return err }, "POST", "/domains/d1/verify", "", "", [2]string{}},
		// API keys
		{"apiKeys.create", func(c *Client) error {
			_, err := c.ApiKeys.Create(ctx, &CreateApiKeyRequest{Name: "Prod", Permission: PermissionSendingAccess, DomainId: "d1"})
			return err
		}, "POST", "/api-keys", "", `{"name":"Prod","permission":"sending_access","domain_id":"d1"}`, [2]string{}},
		{"apiKeys.list", func(c *Client) error { _, err := c.ApiKeys.List(ctx, nil); return err }, "GET", "/api-keys", "", "", [2]string{}},
		{"apiKeys.update", func(c *Client) error {
			_, err := c.ApiKeys.Update(ctx, "k1", &UpdateApiKeyRequest{Name: "New"})
			return err
		}, "PATCH", "/api-keys/k1", "", `{"name":"New"}`, [2]string{}},
		{"apiKeys.remove", func(c *Client) error { _, err := c.ApiKeys.Remove(ctx, "k1"); return err }, "DELETE", "/api-keys/k1", "", "", [2]string{}},
		// Contacts
		{"contacts.create", func(c *Client) error {
			_, err := c.Contacts.Create(ctx, &CreateContactRequest{Email: "ana@x.y", FirstName: "Ana", Properties: map[string]any{"plan": "pro"}, Segments: []Segment{{Id: "s1"}}})
			return err
		}, "POST", "/contacts", "", `{"email":"ana@x.y","first_name":"Ana","properties":{"plan":"pro"},"segments":[{"id":"s1"}]}`, [2]string{}},
		{"contacts.create audience", func(c *Client) error {
			_, err := c.Contacts.Create(ctx, &CreateContactRequest{Email: "ana@x.y", AudienceId: "aud1"})
			return err
		}, "POST", "/audiences/aud1/contacts", "", `{"email":"ana@x.y"}`, [2]string{}},
		{"contacts.list", func(c *Client) error {
			_, err := c.Contacts.List(ctx, &ListContactsParams{ListParams: ListParams{Limit: 100}, SegmentId: "s1"})
			return err
		}, "GET", "/contacts", "limit=100&segment_id=s1", "", [2]string{}},
		{"contacts.list audience", func(c *Client) error {
			_, err := c.Contacts.List(ctx, &ListContactsParams{AudienceId: "a1"})
			return err
		}, "GET", "/contacts", "audience_id=a1", "", [2]string{}},
		{"contacts.list nil", func(c *Client) error { _, err := c.Contacts.List(ctx, nil); return err }, "GET", "/contacts", "", "", [2]string{}},
		{"contacts.get", func(c *Client) error { _, err := c.Contacts.Get(ctx, "ana+x@x.y"); return err }, "GET", "/contacts/ana+x@x.y", "", "", [2]string{}},
		{"contacts.update", func(c *Client) error {
			_, err := c.Contacts.Update(ctx, "ana@x.y", &UpdateContactRequest{Unsubscribed: Bool(true), LastName: "Lima"})
			return err
		}, "PATCH", "/contacts/ana@x.y", "", `{"last_name":"Lima","unsubscribed":true}`, [2]string{}},
		{"contacts.update audience", func(c *Client) error {
			_, err := c.Contacts.Update(ctx, "c1", &UpdateContactRequest{AudienceId: "a1", FirstName: "A"})
			return err
		}, "PATCH", "/audiences/a1/contacts/c1", "", `{"first_name":"A"}`, [2]string{}},
		{"contacts.remove", func(c *Client) error { _, err := c.Contacts.Remove(ctx, "c1"); return err }, "DELETE", "/contacts/c1", "", "", [2]string{}},
		{"contacts.segments.list", func(c *Client) error { _, err := c.Contacts.Segments.List(ctx, "ana@x.y", nil); return err }, "GET", "/contacts/ana@x.y/segments", "", "", [2]string{}},
		{"contacts.segments.add", func(c *Client) error { _, err := c.Contacts.Segments.Add(ctx, "c1", "s1"); return err }, "POST", "/contacts/c1/segments/s1", "", "", [2]string{}},
		{"contacts.segments.remove", func(c *Client) error { _, err := c.Contacts.Segments.Remove(ctx, "c1", "s1"); return err }, "DELETE", "/contacts/c1/segments/s1", "", "", [2]string{}},
		// Segments / audiences
		{"segments.create", func(c *Client) error {
			_, err := c.Segments.Create(ctx, &CreateSegmentRequest{Name: "Pro"})
			return err
		}, "POST", "/segments", "", `{"name":"Pro"}`, [2]string{}},
		{"segments.list", func(c *Client) error { _, err := c.Segments.List(ctx, nil); return err }, "GET", "/segments", "", "", [2]string{}},
		{"segments.get", func(c *Client) error { _, err := c.Segments.Get(ctx, "s1"); return err }, "GET", "/segments/s1", "", "", [2]string{}},
		{"segments.update", func(c *Client) error {
			_, err := c.Segments.Update(ctx, "s1", &UpdateSegmentRequest{Name: "Team"})
			return err
		}, "PATCH", "/segments/s1", "", `{"name":"Team"}`, [2]string{}},
		{"segments.remove", func(c *Client) error { _, err := c.Segments.Remove(ctx, "s1"); return err }, "DELETE", "/segments/s1", "", "", [2]string{}},
		{"segments.listContacts", func(c *Client) error {
			_, err := c.Segments.ListContacts(ctx, "s1", &ListParams{Limit: 50})
			return err
		}, "GET", "/segments/s1/contacts", "limit=50", "", [2]string{}},
		{"audiences alias", func(c *Client) error { _, err := c.Audiences.Get(ctx, "s1"); return err }, "GET", "/segments/s1", "", "", [2]string{}},
		// Broadcasts
		{"broadcasts.create", func(c *Client) error {
			_, err := c.Broadcasts.Create(ctx, &CreateBroadcastRequest{SegmentId: "s1", From: "n@acme.com", Subject: "News", Html: "<p>Hi</p>", PreviewText: "p", Send: true, ScheduledAt: "tomorrow"})
			return err
		}, "POST", "/broadcasts", "", `{"segment_id":"s1","from":"n@acme.com","subject":"News","html":"<p>Hi</p>","preview_text":"p","send":true,"scheduled_at":"tomorrow"}`, [2]string{}},
		{"broadcasts.list", func(c *Client) error { _, err := c.Broadcasts.List(ctx, nil); return err }, "GET", "/broadcasts", "", "", [2]string{}},
		{"broadcasts.get", func(c *Client) error { _, err := c.Broadcasts.Get(ctx, "b1"); return err }, "GET", "/broadcasts/b1", "", "", [2]string{}},
		{"broadcasts.update", func(c *Client) error {
			_, err := c.Broadcasts.Update(ctx, "b1", &UpdateBroadcastRequest{Subject: "New"})
			return err
		}, "PATCH", "/broadcasts/b1", "", `{"subject":"New"}`, [2]string{}},
		{"broadcasts.remove", func(c *Client) error { _, err := c.Broadcasts.Remove(ctx, "b1"); return err }, "DELETE", "/broadcasts/b1", "", "", [2]string{}},
		{"broadcasts.send now", func(c *Client) error { _, err := c.Broadcasts.Send(ctx, "b1", nil); return err }, "POST", "/broadcasts/b1/send", "", `{}`, [2]string{}},
		{"broadcasts.send scheduled", func(c *Client) error {
			_, err := c.Broadcasts.Send(ctx, "b1", &SendBroadcastRequest{ScheduledAt: "in 1 hour"}, WithIdempotencyKey("bk"))
			return err
		}, "POST", "/broadcasts/b1/send", "", `{"scheduled_at":"in 1 hour"}`, [2]string{"Idempotency-Key", "bk"}},
		{"broadcasts.cancel", func(c *Client) error { _, err := c.Broadcasts.Cancel(ctx, "b1"); return err }, "POST", "/broadcasts/b1/cancel", "", "", [2]string{}},
		{"broadcasts.duplicate", func(c *Client) error { _, err := c.Broadcasts.Duplicate(ctx, "b1"); return err }, "POST", "/broadcasts/b1/duplicate", "", "", [2]string{}},
		// Templates
		{"templates.create", func(c *Client) error {
			_, err := c.Templates.Create(ctx, &CreateTemplateRequest{Name: "Welcome", Alias: "welcome", Subject: "Hi {{{NAME}}}", Html: "<p>Hi</p>", Variables: []TemplateVariableInput{{Key: "NAME", Type: "string", FallbackValue: "there"}, {Key: "DAYS", Type: "number", FallbackValue: 14}}})
			return err
		}, "POST", "/templates", "", `{"name":"Welcome","alias":"welcome","subject":"Hi {{{NAME}}}","html":"<p>Hi</p>","variables":[{"key":"NAME","type":"string","fallback_value":"there"},{"key":"DAYS","type":"number","fallback_value":14}]}`, [2]string{}},
		{"templates.list", func(c *Client) error { _, err := c.Templates.List(ctx, nil); return err }, "GET", "/templates", "", "", [2]string{}},
		{"templates.get", func(c *Client) error { _, err := c.Templates.Get(ctx, "welcome"); return err }, "GET", "/templates/welcome", "", "", [2]string{}},
		{"templates.update", func(c *Client) error {
			_, err := c.Templates.Update(ctx, "welcome", &UpdateTemplateRequest{Subject: "New"})
			return err
		}, "PATCH", "/templates/welcome", "", `{"subject":"New"}`, [2]string{}},
		{"templates.remove", func(c *Client) error { _, err := c.Templates.Remove(ctx, "welcome"); return err }, "DELETE", "/templates/welcome", "", "", [2]string{}},
		{"templates.publish", func(c *Client) error { _, err := c.Templates.Publish(ctx, "welcome"); return err }, "POST", "/templates/welcome/publish", "", "", [2]string{}},
		{"templates.duplicate", func(c *Client) error { _, err := c.Templates.Duplicate(ctx, "welcome"); return err }, "POST", "/templates/welcome/duplicate", "", "", [2]string{}},
		// Webhooks
		{"webhooks.create", func(c *Client) error {
			_, err := c.Webhooks.Create(ctx, &CreateWebhookRequest{Endpoint: "https://acme.com/wh", Events: []string{EventEmailDelivered, EventEmailBounced}})
			return err
		}, "POST", "/webhooks", "", `{"endpoint":"https://acme.com/wh","events":["email.delivered","email.bounced"]}`, [2]string{}},
		{"webhooks.list", func(c *Client) error { _, err := c.Webhooks.List(ctx, nil); return err }, "GET", "/webhooks", "", "", [2]string{}},
		{"webhooks.get", func(c *Client) error { _, err := c.Webhooks.Get(ctx, "w1"); return err }, "GET", "/webhooks/w1", "", "", [2]string{}},
		{"webhooks.update", func(c *Client) error {
			_, err := c.Webhooks.Update(ctx, "w1", &UpdateWebhookRequest{Status: "disabled"})
			return err
		}, "PATCH", "/webhooks/w1", "", `{"status":"disabled"}`, [2]string{}},
		{"webhooks.remove", func(c *Client) error { _, err := c.Webhooks.Remove(ctx, "w1"); return err }, "DELETE", "/webhooks/w1", "", "", [2]string{}},
		{"webhooks.rotate", func(c *Client) error { _, err := c.Webhooks.RotateSigningSecret(ctx, "w1"); return err }, "POST", "/webhooks/w1/signing-secret/rotate", "", "", [2]string{}},
		// Suppressions
		{"suppressions.add", func(c *Client) error {
			_, err := c.Suppressions.Add(ctx, &AddSuppressionRequest{Email: "x@y.z"})
			return err
		}, "POST", "/suppressions", "", `{"email":"x@y.z"}`, [2]string{}},
		{"suppressions.list", func(c *Client) error {
			_, err := c.Suppressions.List(ctx, &ListSuppressionsParams{Origin: OriginBounce})
			return err
		}, "GET", "/suppressions", "origin=bounce", "", [2]string{}},
		{"suppressions.list nil", func(c *Client) error { _, err := c.Suppressions.List(ctx, nil); return err }, "GET", "/suppressions", "", "", [2]string{}},
		{"suppressions.get", func(c *Client) error { _, err := c.Suppressions.Get(ctx, "x@y.z"); return err }, "GET", "/suppressions/x@y.z", "", "", [2]string{}},
		{"suppressions.remove", func(c *Client) error { _, err := c.Suppressions.Remove(ctx, "x@y.z"); return err }, "DELETE", "/suppressions/x@y.z", "", "", [2]string{}},
		{"suppressions.batch.add", func(c *Client) error {
			_, err := c.Suppressions.Batch.Add(ctx, &AddSuppressionBatchRequest{Emails: []string{"a@x.y", "b@x.y"}})
			return err
		}, "POST", "/suppressions/batch/add", "", `{"emails":["a@x.y","b@x.y"]}`, [2]string{}},
		{"suppressions.batch.remove", func(c *Client) error {
			_, err := c.Suppressions.Batch.Remove(ctx, &RemoveSuppressionBatchRequest{Ids: []string{"i1"}})
			return err
		}, "POST", "/suppressions/batch/remove", "", `{"ids":["i1"]}`, [2]string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api, c := newAPI(t)
			if err := tc.call(c); err != nil {
				t.Fatal(err)
			}
			r := api.last()
			if r.Method != tc.method || r.Path != tc.path || r.Query != tc.query {
				t.Fatalf("got %s %s?%s, want %s %s?%s", r.Method, r.Path, r.Query, tc.method, tc.path, tc.query)
			}
			if tc.body == "" {
				if len(r.Body) != 0 {
					t.Fatalf("unexpected body %s", r.Body)
				}
			} else {
				mustEqualJSON(t, r.JSON(t), tc.body)
			}
			if tc.header[0] != "" && r.Header.Get(tc.header[0]) != tc.header[1] {
				t.Fatalf("header %s = %q", tc.header[0], r.Header.Get(tc.header[0]))
			}
		})
	}
}

func TestRequiredIDs(t *testing.T) {
	ctx := context.Background()
	api, c := newAPI(t)
	calls := []func() error{
		func() error { _, err := c.Emails.Update(ctx, "", &UpdateEmailRequest{}); return err },
		func() error { _, err := c.Emails.Cancel(ctx, ""); return err },
		func() error { _, err := c.Emails.Attachments.List(ctx, "", nil); return err },
		func() error { _, err := c.Emails.Attachments.Get(ctx, "", "a"); return err },
		func() error { _, err := c.Emails.Attachments.Get(ctx, "e", ""); return err },
		func() error { _, err := c.Domains.Get(ctx, ""); return err },
		func() error { _, err := c.Domains.Update(ctx, "", nil); return err },
		func() error { _, err := c.Domains.Remove(ctx, ""); return err },
		func() error { _, err := c.Domains.Verify(ctx, ""); return err },
		func() error { _, err := c.ApiKeys.Update(ctx, "", nil); return err },
		func() error { _, err := c.ApiKeys.Remove(ctx, ""); return err },
		func() error { _, err := c.Contacts.Get(ctx, ""); return err },
		func() error { _, err := c.Contacts.Update(ctx, "", nil); return err },
		func() error { _, err := c.Contacts.Remove(ctx, ""); return err },
		func() error { _, err := c.Contacts.Segments.List(ctx, "", nil); return err },
		func() error { _, err := c.Contacts.Segments.Add(ctx, "", "s"); return err },
		func() error { _, err := c.Contacts.Segments.Add(ctx, "c", ""); return err },
		func() error { _, err := c.Contacts.Segments.Remove(ctx, "", "s"); return err },
		func() error { _, err := c.Contacts.Segments.Remove(ctx, "c", ""); return err },
		func() error { _, err := c.Segments.Get(ctx, ""); return err },
		func() error { _, err := c.Segments.Update(ctx, "", nil); return err },
		func() error { _, err := c.Segments.Remove(ctx, ""); return err },
		func() error { _, err := c.Segments.ListContacts(ctx, "", nil); return err },
		func() error { _, err := c.Broadcasts.Get(ctx, ""); return err },
		func() error { _, err := c.Broadcasts.Update(ctx, "", nil); return err },
		func() error { _, err := c.Broadcasts.Remove(ctx, ""); return err },
		func() error { _, err := c.Broadcasts.Send(ctx, "", nil); return err },
		func() error { _, err := c.Broadcasts.Cancel(ctx, ""); return err },
		func() error { _, err := c.Broadcasts.Duplicate(ctx, ""); return err },
		func() error { _, err := c.Templates.Get(ctx, ""); return err },
		func() error { _, err := c.Templates.Update(ctx, "", nil); return err },
		func() error { _, err := c.Templates.Remove(ctx, ""); return err },
		func() error { _, err := c.Templates.Publish(ctx, ""); return err },
		func() error { _, err := c.Templates.Duplicate(ctx, ""); return err },
		func() error { _, err := c.Webhooks.Get(ctx, ""); return err },
		func() error { _, err := c.Webhooks.Update(ctx, "", nil); return err },
		func() error { _, err := c.Webhooks.Remove(ctx, ""); return err },
		func() error { _, err := c.Webhooks.RotateSigningSecret(ctx, ""); return err },
		func() error { _, err := c.Suppressions.Get(ctx, ""); return err },
		func() error { _, err := c.Suppressions.Remove(ctx, ""); return err },
	}
	for i, call := range calls {
		e, ok := AsError(call())
		if !ok || e.Name != ErrMissingRequiredField {
			t.Fatalf("call %d: err = %v", i, e)
		}
	}
	if api.count() != 0 {
		t.Fatalf("%d requests sent", api.count())
	}
}

func TestDecodeTypedResponses(t *testing.T) {
	ctx := context.Background()
	_, c := newAPI(t,
		reply{body: `{"object":"domain","id":"d1","name":"acme.com","status":"pending","tls":"opportunistic","capabilities":{"sending":"enabled","receiving":"disabled"},"records":[{"record":"DKIM","name":"pidgely._domainkey","type":"TXT","ttl":"Auto","status":"pending","value":"p=MIG"},{"record":"SPF","name":"bounces","type":"MX","ttl":"Auto","status":"pending","value":"feedback.pidgely.com","priority":10}]}`},
		reply{body: `{"object":"list","has_more":true,"data":[{"id":"c1","email":"a@b.c","unsubscribed":false,"properties":{"plan":{"value":"pro","type":"string"}}}]}`},
		reply{body: `{"object":"broadcast","id":"b1","status":"sent","stats":{"delivered":10,"bounced":1},"recipients":11,"segment_id":"s1","audience_id":null}`},
		reply{body: `{"object":"template","id":"t1","alias":"welcome","status":"published","variables":[{"key":"N","type":"number","fallback_value":14}],"has_unpublished_versions":false}`},
		reply{body: `{"object":"api_key","id":"k1","token":"pdg_abc_secret"}`},
		reply{body: `{"object":"webhook","id":"w1","signing_secret":"whsec_abc"}`},
		reply{body: `{"data":[{"id":"e1"}],"errors":[{"index":1,"message":"bad to"}]}`},
		reply{body: `{"data":[{"object":"suppression","id":"s1","deleted":true}]}`},
	)
	d, err := c.Domains.Create(ctx, &CreateDomainRequest{Name: "acme.com"})
	if err != nil || len(d.Records) != 2 || d.Records[1].Priority != 10 || d.Capabilities.Sending != "enabled" {
		t.Fatalf("%+v %v", d, err)
	}
	cs, err := c.Contacts.List(ctx, nil)
	if err != nil || !cs.HasMore || cs.Data[0].Properties["plan"].Value != "pro" {
		t.Fatalf("%+v %v", cs, err)
	}
	b, err := c.Broadcasts.Get(ctx, "b1")
	if err != nil || b.Stats["delivered"] != 10 || b.Recipients != 11 || b.AudienceId != "" {
		t.Fatalf("%+v %v", b, err)
	}
	tpl, err := c.Templates.Get(ctx, "welcome")
	if err != nil || tpl.Variables[0].FallbackValue != float64(14) {
		t.Fatalf("%+v %v", tpl, err)
	}
	k, err := c.ApiKeys.Create(ctx, &CreateApiKeyRequest{Name: "x"})
	if err != nil || k.Token != "pdg_abc_secret" {
		t.Fatalf("%+v %v", k, err)
	}
	w, err := c.Webhooks.Create(ctx, &CreateWebhookRequest{Endpoint: "https://x", Events: []string{EventEmailSent}})
	if err != nil || w.SigningSecret != "whsec_abc" {
		t.Fatalf("%+v %v", w, err)
	}
	batch, err := c.Batch.Send(ctx, []SendEmailRequest{{To: []string{"a"}}, {To: []string{"b"}}}, WithBatchValidation(BatchPermissive))
	if err != nil || batch.Data[0].Id != "e1" || batch.Errors[0].Index != 1 {
		t.Fatalf("%+v %v", batch, err)
	}
	rm, err := c.Suppressions.Batch.Remove(ctx, &RemoveSuppressionBatchRequest{Emails: []string{"a@b.c"}})
	if err != nil || !rm.Data[0].Deleted {
		t.Fatalf("%+v %v", rm, err)
	}
}

func TestWebhooksVerifyShortcut(t *testing.T) {
	c := NewClient("k")
	h := http.Header{}
	h.Set("svix-id", vectorID)
	h.Set("svix-timestamp", "1614265330")
	h.Set("svix-signature", vectorSignature)
	if err := c.Webhooks.Verify([]byte(vectorPayload), h, vectorSecret, vectorNow()); err != nil {
		t.Fatal(err)
	}
}

// Official Standard Webhooks / Svix test vector.
const (
	vectorSecret    = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"
	vectorID        = "msg_p5jXN8AQM9LWM0D4loKWxJek"
	vectorPayload   = `{"test": 2432232314}`
	vectorSignature = "v1,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE="
)

func vectorNow() webhook.Option {
	return webhook.WithNow(func() time.Time { return time.Unix(1614265330, 0) })
}

func TestPointerHelpers(t *testing.T) {
	if !*Bool(true) || *String("x") != "x" {
		t.Fatal("helpers")
	}
	var a Audience = Segment{Id: "s"}
	_ = CreateAudienceRequest{Name: a.Id}
}
