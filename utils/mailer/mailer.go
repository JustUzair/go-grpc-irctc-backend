package mailer

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"strings"

	"github.com/JustUzair/irctc-microservice/utils"
	custom_errors "github.com/JustUzair/irctc-microservice/utils/errors"
	"github.com/resend/resend-go/v3"
)

type ResendConfig struct {
	APIKey      string
	FromName    string
	FromAddress string
}

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
	renderer    *TemplateRenderer
}

type TemplateRenderer struct {
	templates *htmltemplate.Template
}

type RenderedEmail struct {
	Subject string
	HTML    string
}

func NewResend(config ResendConfig) (*ResendMailer, error) {

	if !utils.IsEmailValid(config.FromAddress) || len(config.FromName) == 0 || len(config.APIKey) == 0 {
		return nil, custom_errors.ERR_INVALID_CONFIG
	}

	renderer, err := NewTemplateRenderer()
	if err != nil {
		return nil, err
	}

	return &ResendMailer{
		client:      resend.NewClient(config.APIKey),
		fromName:    config.FromName,
		fromAddress: config.FromAddress,
		renderer:    renderer,
	}, nil
}

func NewTemplateRenderer() (*TemplateRenderer, error) {
	parsedTemplates, err := htmltemplate.New("emails").
		Option("missingkey=error").
		ParseFS(emailTemplateFiles, "templates/*.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse email templates: %w", err)
	}

	return &TemplateRenderer{templates: parsedTemplates}, nil
}

func (m *ResendMailer) SendEmail(ctx context.Context, emailTemplate EmailTemplate, params EmailParams) (string, error) {
	if !utils.IsEmailValid(params.ToEmailAddress) {
		return "", fmt.Errorf("invalid recipient email address")
	}

	rendered, err := m.renderer.Render(emailTemplate, params.TemplateData)
	if err != nil {
		return "", err
	}

	request := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", m.fromName, m.fromAddress),
		To:      []string{params.ToEmailAddress},
		Subject: rendered.Subject,
		Html:    rendered.HTML,
	}

	sent, err := m.client.Emails.SendWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf("send email through Resend: %w", err)
	}

	return sent.Id, nil
}

func (r *TemplateRenderer) Render(emailTemplate EmailTemplate, templateData any) (RenderedEmail, error) {
	var (
		subject      string
		templateName string
		data         any
	)

	switch emailTemplate {
	case SendOTP:
		sendOTPData, ok := templateData.(SendOTPTemplateData)
		if !ok {
			return RenderedEmail{}, fmt.Errorf("invalid data for send OTP template")
		}
		if strings.TrimSpace(sendOTPData.OTP) == "" || sendOTPData.ExpiresInMinutes <= 0 {
			return RenderedEmail{}, fmt.Errorf("OTP and expiration are required")
		}

		subject = sendOTPSubject
		templateName = sendOTPTemplateName
		data = sendOTPData

	case VerifyOTP:
		verifyOTPData, ok := templateData.(VerifyOTPTemplateData)
		if !ok {
			return RenderedEmail{}, fmt.Errorf("invalid data for verify OTP template")
		}

		subject = verifyOTPSubject
		templateName = verifyOTPTemplateName
		data = verifyOTPData
	default:
		return RenderedEmail{}, custom_errors.ERR_INVALID_TEMPLATE
	}

	htmlBody, err := r.renderTemplate(templateName, data)
	if err != nil {
		return RenderedEmail{}, err
	}

	return RenderedEmail{Subject: subject, HTML: htmlBody}, nil
}

func (r *TemplateRenderer) renderTemplate(templateName string, data any) (string, error) {
	var body bytes.Buffer

	if err := r.templates.ExecuteTemplate(&body, templateName, data); err != nil {
		return "", fmt.Errorf("render email template %q: %w", templateName, err)
	}

	return body.String(), nil
}
