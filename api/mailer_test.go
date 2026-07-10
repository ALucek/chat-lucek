package main

import (
	"context"
	"testing"
)

func TestFakeMailer_CapturesLink(t *testing.T) {
	m := newFakeMailer()
	_ = m.SendMagicLink(context.Background(), "a@b.com", "https://x/auth/magic?token=t")
	got, ok := m.last("a@b.com")
	if !ok || got != "https://x/auth/magic?token=t" {
		t.Fatalf("capture failed: %q %v", got, ok)
	}
}

func TestSelectMailer_FakeWhenNoKey(t *testing.T) {
	if _, ok := selectMailer(Config{}).(*fakeMailer); !ok {
		t.Fatal("empty RESEND_API_KEY should select the fake mailer")
	}
}

func TestSelectMailer_ResendWhenKeySet(t *testing.T) {
	m := selectMailer(Config{ResendAPIKey: "k", MagicLinkFrom: "login@lucek.ai"})
	if _, ok := m.(*resendMailer); !ok {
		t.Fatal("a set RESEND_API_KEY should select the real mailer")
	}
}
