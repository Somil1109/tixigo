package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"

	"github.com/tixigo/tixigo-api/internal/config"
)

type EmailSender interface {
	Send(context.Context, string, string, string) error
}

func NewEmailSender(cfg config.Config) EmailSender {
	if cfg.ResendAPIKey != "" {
		return &resendSender{cfg.ResendAPIKey, cfg.EmailFrom, http.DefaultClient}
	}
	return &smtpSender{cfg.MailpitAddress, cfg.EmailFrom}
}

type resendSender struct {
	apiKey, from string
	client       *http.Client
}

func (s *resendSender) Send(ctx context.Context, to, subject, html string) error {
	body, _ := json.Marshal(map[string]any{"from": s.from, "to": []string{to}, "subject": subject, "html": html})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Resend returned %s", res.Status)
	}
	return nil
}

type smtpSender struct{ address, from string }

func (s *smtpSender) Send(_ context.Context, to, subject, html string) error {
	from, err := mail.ParseAddress(s.from)
	if err != nil {
		return err
	}
	message := strings.Join([]string{"From: " + s.from, "To: " + to, "Subject: " + subject, "MIME-Version: 1.0", "Content-Type: text/html; charset=UTF-8", "", html}, "\r\n")
	return smtp.SendMail(s.address, nil, from.Address, []string{to}, []byte(message))
}
