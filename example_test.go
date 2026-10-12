package pidgely_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/pidgely/pidgely-go"
	"github.com/pidgely/pidgely-go/webhook"
)

func ExampleNewClient() {
	client := pidgely.NewClient(os.Getenv("PIDGELY_API_KEY"))

	sent, err := client.Emails.Send(context.Background(), &pidgely.SendEmailRequest{
		From:    "Acme <hello@acme.com>",
		To:      []string{"ana@example.org"},
		Subject: "Welcome to Acme",
		Html:    "<p>Your account is ready.</p>",
	}, pidgely.WithIdempotencyKey("welcome-ana"))
	if err != nil {
		var apiErr *pidgely.Error
		if errors.As(err, &apiErr) {
			log.Printf("%s (%d): %s", apiErr.Name, apiErr.StatusCode, apiErr.Message)
		}
		return
	}
	fmt.Println(sent.Id)
}

func ExampleTemplatesService_Publish() {
	client := pidgely.NewClient("")
	ctx := context.Background()

	created, err := client.Templates.Create(ctx, &pidgely.CreateTemplateRequest{
		Name:    "Welcome",
		Alias:   "welcome",
		Subject: "Welcome, {{{NAME}}}!",
		Html:    "<p>Hi {{{NAME}}}</p>",
		Variables: []pidgely.TemplateVariableInput{
			{Key: "NAME", Type: "string", FallbackValue: "there"},
		},
	})
	if err != nil {
		return
	}
	_, _ = client.Templates.Publish(ctx, created.Id)
}

func ExampleWebhooksService_Verify() {
	http.HandleFunc("/webhooks/pidgely", func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		event, err := webhook.Parse(payload, r.Header, os.Getenv("PIDGELY_WEBHOOK_SECRET"))
		if err != nil {
			http.Error(w, "invalid signature", http.StatusBadRequest)
			return
		}
		log.Println(event.Type)
		w.WriteHeader(http.StatusNoContent)
	})
}
