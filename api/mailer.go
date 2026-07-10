package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/resend/resend-go/v2"
)

// mailer sends a magic-link email to an address.
type mailer interface {
	SendMagicLink(ctx context.Context, to, link string) error
}

// fakeMailer captures links in memory instead of sending. Tests + local dev.
type fakeMailer struct {
	mu    sync.Mutex
	links map[string]string
}

func newFakeMailer() *fakeMailer { return &fakeMailer{links: map[string]string{}} }

func (f *fakeMailer) SendMagicLink(_ context.Context, to, link string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.links[to] = link
	slog.Info("magic link (fake mailer)", "to", to, "link", link)
	return nil
}

func (f *fakeMailer) last(to string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.links[to]
	return l, ok
}

// resendMailer sends through Resend's official Go SDK.
type resendMailer struct {
	client *resend.Client
	from   string
}

func (m *resendMailer) SendMagicLink(ctx context.Context, to, link string) error {
	_, err := m.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: "Your sign-in link",
		Html:    fmt.Sprintf(`<p>Click to sign in:</p><p><a href="%s">%s</a></p><p>This link expires in 15 minutes. Ignore this email if you did not request it.</p>`, link, link),
		Text:    fmt.Sprintf("Sign in: %s\n\nThis link expires in 15 minutes. Ignore this email if you did not request it.", link),
	})
	return err
}

// selectMailer: fake mailer when RESEND_API_KEY empty, else real Resend.
func selectMailer(cfg Config) mailer {
	if cfg.ResendAPIKey == "" {
		slog.Warn("RESEND_API_KEY empty: using fake mailer (logs links) — never in production")
		return newFakeMailer()
	}
	return &resendMailer{client: resend.NewClient(cfg.ResendAPIKey), from: cfg.MagicLinkFrom}
}
