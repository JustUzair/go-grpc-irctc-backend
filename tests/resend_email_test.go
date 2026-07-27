//go:build integration && resend

package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/mailer"
)

const resendDeliveredTestAddress = "delivered+irctc-integration@resend.dev"

func TestResendEmailIntegration(t *testing.T) {
	config, err := env.Load()
	if err != nil {
		t.Fatalf("load email test configuration: %v", err)
	}
	if strings.TrimSpace(config.ResendAPIKey) == "" {
		t.Skip("RESEND_API_KEY is not configured; skipping live email integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)

	t.Log("initializing Resend mailer")
	emailClient, err := mailer.NewResend()
	if err != nil {
		t.Fatalf("initialize Resend mailer: %v", err)
	}

	t.Log("sending OTP template to Resend test recipient")
	messageID, err := emailClient.SendEmail(ctx, mailer.SendOTP, mailer.EmailParams{
		ToEmailAddress: resendDeliveredTestAddress,
		TemplateData: mailer.SendOTPTemplateData{
			Name:             "Integration Test User",
			OTP:              "123456",
			ExpiresInMinutes: 10,
		},
	})
	if err != nil {
		t.Fatalf("send test email: %v", err)
	}
	if strings.TrimSpace(messageID) == "" {
		t.Fatal("Resend returned an empty message ID")
	}

	t.Logf("Resend accepted test email with message ID %s", messageID)
}
