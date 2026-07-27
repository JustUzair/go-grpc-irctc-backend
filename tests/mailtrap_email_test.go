//go:build integration && mailtrap

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JustUzair/irctc-microservice/utils/env"
	"github.com/JustUzair/irctc-microservice/utils/mailer"
)

const mailtrapSandboxEndpoint = "https://sandbox.api.mailtrap.io/api/send/%d"

type mailtrapAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type mailtrapSendRequest struct {
	From     mailtrapAddress   `json:"from"`
	To       []mailtrapAddress `json:"to"`
	Subject  string            `json:"subject"`
	HTML     string            `json:"html"`
	Category string            `json:"category"`
}

type mailtrapSendResponse struct {
	Success    bool     `json:"success"`
	MessageIDs []string `json:"message_ids"`
}

func TestMailtrapEmailIntegration(t *testing.T) {
	config, err := env.Load()
	if err != nil {
		t.Fatalf("load email test configuration: %v", err)
	}

	apiToken := strings.TrimSpace(os.Getenv("MAILTRAP_API_TOKEN"))
	if apiToken == "" {
		apiToken = strings.TrimSpace(os.Getenv("MAILTRAP_TOKEN"))
	}
	inboxIDValue := strings.TrimSpace(os.Getenv("MAILTRAP_INBOX_ID"))
	recipient := strings.TrimSpace(os.Getenv("MAILTRAP_TEST_RECIPIENT"))
	if apiToken == "" || inboxIDValue == "" {
		t.Skip("MAILTRAP_API_TOKEN and MAILTRAP_INBOX_ID are required")
	}
	if recipient == "" {
		recipient = "integration@example.com"
	}

	inboxID, err := strconv.ParseInt(inboxIDValue, 10, 64)
	if err != nil || inboxID <= 0 {
		t.Fatalf("MAILTRAP_INBOX_ID must be a positive integer")
	}

	t.Log("rendering OTP email with shared mailer templates")
	renderer, err := mailer.NewTemplateRenderer()
	if err != nil {
		t.Fatalf("initialize email template renderer: %v", err)
	}
	rendered, err := renderer.Render(mailer.SendOTP, mailer.SendOTPTemplateData{
		Name:             "Integration Test User",
		OTP:              "123456",
		ExpiresInMinutes: 10,
	})
	if err != nil {
		t.Fatalf("render OTP email: %v", err)
	}

	payload, err := json.Marshal(mailtrapSendRequest{
		From: mailtrapAddress{
			Email: config.EmailFromAddress,
			Name:  config.EmailFromName,
		},
		To:       []mailtrapAddress{{Email: recipient}},
		Subject:  rendered.Subject,
		HTML:     rendered.HTML,
		Category: "Integration Test",
	})
	if err != nil {
		t.Fatalf("encode Mailtrap request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)

	endpoint := fmt.Sprintf(mailtrapSandboxEndpoint, inboxID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create Mailtrap request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+apiToken)
	request.Header.Set("Content-Type", "application/json")

	t.Log("sending rendered OTP email to Mailtrap sandbox")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send email to Mailtrap sandbox: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		t.Fatalf("read Mailtrap response: %v", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("Mailtrap returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}

	var result mailtrapSendResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		t.Fatalf("decode Mailtrap response: %v", err)
	}
	if !result.Success || len(result.MessageIDs) == 0 {
		t.Fatalf("Mailtrap did not accept the test email")
	}

	t.Logf("Mailtrap accepted rendered email with message ID %s", result.MessageIDs[0])
}
