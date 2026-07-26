package mailer

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"log"
	"strings"

	"github.com/JustUzair/irctc-microservice/utils"
	"github.com/JustUzair/irctc-microservice/utils/env"
	custom_errors "github.com/JustUzair/irctc-microservice/utils/errors"
	"github.com/resend/resend-go/v3"
)

//go:embed templates/*.html.tmpl
var emailTemplateFiles embed.FS

type EmailParams struct {
	ToEmailAddress string
	TemplateData   any
}

type SendOTPTemplateData struct {
	Name             string
	OTP              string
	ExpiresInMinutes int
}

type VerifyOTPTemplateData struct {
	Name string
}

type EmailTemplate int

const (
	SendOTP EmailTemplate = iota
	VerifyOTP

	sendOTPTemplateName   = "send-otp.html.tmpl"
	verifyOTPTemplateName = "verify-otp.html.tmpl"
	sendOTPSubject        = "Verify your email address"
	verifyOTPSubject      = "Your email has been verified"
)

type ResendMailer struct {
	client      *resend.Client
	fromName    string
	fromAddress string
	templates   *htmltemplate.Template
}

func NewResend() (*ResendMailer, error) {
	// Validate configuration first.
	config, err := env.Load()
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		return nil, custom_errors.ERR_INVALID_CONFIG
	}
	if !utils.IsEmailValid(config.EmailFromAddress) || len(config.EmailFromName) == 0 || len(config.ResendAPIKey) == 0 {
		return nil, custom_errors.ERR_INVALID_CONFIG
	}

	parsedTemplates, err := htmltemplate.New("emails").
		Option("missingkey=error").
		ParseFS(emailTemplateFiles, "templates/*.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse email templates: %w", err)
	}

	return &ResendMailer{
		client:      resend.NewClient(config.ResendAPIKey),
		fromName:    config.EmailFromName,
		fromAddress: config.EmailFromAddress,
		templates:   parsedTemplates,
	}, nil
}

func (m *ResendMailer) SendEmail(ctx context.Context, emailTemplate EmailTemplate, params EmailParams) (string, error) {
	if !utils.IsEmailValid(params.ToEmailAddress) {
		return "", fmt.Errorf("invalid recipient email address")
	}

	var (
		subject      string
		templateName string
		templateData any
	)

	switch emailTemplate {
	case SendOTP:
		data, ok := params.TemplateData.(SendOTPTemplateData)
		if !ok {
			return "", fmt.Errorf("invalid data for send OTP template")
		}
		if strings.TrimSpace(data.OTP) == "" || data.ExpiresInMinutes <= 0 {
			return "", fmt.Errorf("OTP and expiration are required")
		}

		subject = sendOTPSubject
		templateName = sendOTPTemplateName
		templateData = data

	case VerifyOTP:
		data, ok := params.TemplateData.(VerifyOTPTemplateData)
		if !ok {
			return "", fmt.Errorf("invalid data for verify OTP template")
		}

		subject = verifyOTPSubject
		templateName = verifyOTPTemplateName
		templateData = data
	default:
		return "", custom_errors.ERR_INVALID_TEMPLATE
	}

	htmlBody, err := m.renderTemplate(templateName, templateData)
	if err != nil {
		return "", err
	}

	request := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", m.fromName, m.fromAddress),
		To:      []string{params.ToEmailAddress},
		Subject: subject,
		Html:    htmlBody,
	}

	sent, err := m.client.Emails.SendWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf("send email through Resend: %w", err)
	}

	return sent.Id, nil
}

func (m *ResendMailer) renderTemplate(templateName string, data any) (string, error) {
	var body bytes.Buffer

	if err := m.templates.ExecuteTemplate(&body, templateName, data); err != nil {
		return "", fmt.Errorf("render email template %q: %w", templateName, err)
	}

	return body.String(), nil
}
