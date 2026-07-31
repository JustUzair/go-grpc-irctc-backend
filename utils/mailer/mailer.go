package mailer

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/JustUzair/go-grpc-irctc-backend/utils"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	custom_errors "github.com/JustUzair/go-grpc-irctc-backend/utils/errors"
	"github.com/resend/resend-go/v3"
)

const mailtrapSandboxEndpoint = "https://sandbox.api.mailtrap.io/api/send/%d"

type ResendConfig struct {
	APIKey      string
	FromName    string
	FromAddress string
}

type MailtrapConfig struct {
	APIToken    string
	InboxID     string
	FromName    string
	FromAddress string
}

type Mailer interface {
	SendEmail(ctx context.Context, emailTemplate EmailTemplate, params EmailParams) (string, error)
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

type MailtrapMailer struct {
	apiToken    string
	inboxID     int64
	fromName    string
	fromAddress string
	renderer    *TemplateRenderer
	client      *http.Client
}

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

func New(config env.Config) (Mailer, error) {
	switch strings.ToLower(strings.TrimSpace(config.MailerProvider)) {
	case "", "resend":
		return NewResend(ResendConfig{
			APIKey:      config.ResendAPIKey,
			FromName:    config.EmailFromName,
			FromAddress: config.EmailFromAddress,
		})
	case "mailtrap":
		return NewMailtrap(MailtrapConfig{
			APIToken:    config.MailtrapAPIToken,
			InboxID:     config.MailtrapInboxID,
			FromName:    config.EmailFromName,
			FromAddress: config.EmailFromAddress,
		})
	default:
		return nil, fmt.Errorf("unsupported mailer provider %q", config.MailerProvider)
	}
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

func NewMailtrap(config MailtrapConfig) (*MailtrapMailer, error) {
	if !utils.IsEmailValid(config.FromAddress) || len(config.FromName) == 0 || len(config.APIToken) == 0 {
		return nil, custom_errors.ERR_INVALID_CONFIG
	}

	inboxID, err := strconv.ParseInt(strings.TrimSpace(config.InboxID), 10, 64)
	if err != nil || inboxID <= 0 {
		return nil, custom_errors.ERR_INVALID_CONFIG
	}

	renderer, err := NewTemplateRenderer()
	if err != nil {
		return nil, err
	}

	return &MailtrapMailer{
		apiToken:    config.APIToken,
		inboxID:     inboxID,
		fromName:    config.FromName,
		fromAddress: config.FromAddress,
		renderer:    renderer,
		client:      http.DefaultClient,
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

func (m *MailtrapMailer) SendEmail(ctx context.Context, emailTemplate EmailTemplate, params EmailParams) (string, error) {
	if !utils.IsEmailValid(params.ToEmailAddress) {
		return "", fmt.Errorf("invalid recipient email address")
	}

	rendered, err := m.renderer.Render(emailTemplate, params.TemplateData)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(mailtrapSendRequest{
		From: mailtrapAddress{
			Email: m.fromAddress,
			Name:  m.fromName,
		},
		To:       []mailtrapAddress{{Email: params.ToEmailAddress}},
		Subject:  rendered.Subject,
		HTML:     rendered.HTML,
		Category: "Transactional",
	})
	if err != nil {
		return "", fmt.Errorf("encode Mailtrap request: %w", err)
	}

	endpoint := fmt.Sprintf(mailtrapSandboxEndpoint, m.inboxID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create Mailtrap request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+m.apiToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := m.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("send email through Mailtrap: %w", err)
	}
	defer response.Body.Close()

	var result mailtrapSendResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode Mailtrap response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("send email through Mailtrap: status %s", response.Status)
	}
	if !result.Success || len(result.MessageIDs) == 0 {
		return "", fmt.Errorf("send email through Mailtrap: no message id returned")
	}

	return result.MessageIDs[0], nil
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
