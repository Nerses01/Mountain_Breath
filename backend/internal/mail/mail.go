// Package mail is the one place the application sends email from.
//
// The interface is deliberately one method, declared here at the point of
// use (the api layer consumes it, exactly like its store interfaces): a
// handler needs "deliver this message" and has no business knowing whether
// that means an SMTP conversation, a provider's REST API (a Phase 9+
// decision, once hosting exists), or a test's slice of sent messages.
package mail

import (
	"context"
	"fmt"
	"log/slog"
	"mime"
	"net/smtp"
	"strings"
)

type Message struct {
	To      string
	Subject string
	// Plain text only, on purpose. HTML mail needs a multipart body, inline
	// CSS and a second copy of every sentence; a password-reset link and an
	// order receipt are exactly as useful in text, land better with spam
	// filters, and read fine in every client since 1982.
	Text string
}

type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// ── SMTP ──────────────────────────────────────────────────────────────────

// SMTP delivers through a relay — Mailpit's catcher in dev (localhost:1025,
// browse the mailbox at :8025), the family's real relay in prod. Auth is
// optional because the dev catcher has none.
type SMTP struct {
	Addr     string // host:port
	From     string // "Mountain Breath <hive@example.com>"
	Username string
	Password string
}

// Send speaks just enough SMTP. Two encoding details matter, because SMTP
// predates UTF-8 by a decade:
//
//   - the Subject must be RFC 2047-encoded or an Armenian "Վերականգնել…"
//     arrives as mojibake — mime.QEncoding does the =?utf-8?q?…?= dance;
//   - the body carries an explicit text/plain; charset=utf-8 header, since
//     the assumed default is US-ASCII.
//
// net/smtp takes no context; the practical timeout is the server's dial
// default. Fine for a dev catcher and a LAN relay — revisit if a slow
// provider ever holds a checkout hostage (the checkout already treats mail
// as non-fatal, which is the real protection).
func (s *SMTP) Send(_ context.Context, m Message) error {
	headers := []string{
		"From: " + s.From,
		"To: " + m.To,
		"Subject: " + mime.QEncoding.Encode("utf-8", m.Subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
	}
	// SMTP line endings are CRLF, and the blank line separates headers from
	// body — both are protocol, not style.
	body := strings.Join(headers, "\r\n") + "\r\n\r\n" + m.Text

	var auth smtp.Auth
	if s.Username != "" {
		host := s.Addr
		if i := strings.LastIndex(host, ":"); i >= 0 {
			host = host[:i]
		}
		auth = smtp.PlainAuth("", s.Username, s.Password, host)
	}
	if err := smtp.SendMail(s.Addr, auth, fromAddress(s.From), []string{m.To}, []byte(body)); err != nil {
		return fmt.Errorf("sending mail to %s: %w", m.To, err)
	}
	return nil
}

// fromAddress extracts the bare address from "Name <addr>" for the SMTP
// envelope — the envelope sender and the From header are different fields,
// and the envelope wants no display name.
func fromAddress(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		return strings.TrimRight(from[i+1:], ">")
	}
	return from
}

// ── The no-configuration fallback ─────────────────────────────────────────

// LogSink is what runs when MB_SMTP_ADDR is unset: the message lands in the
// server log instead of a mailbox, so the app works out of the box and a
// developer still SEES what would have been sent. It is not a dev/prod
// switch — dev normally uses Mailpit, which is real SMTP.
type LogSink struct {
	Log *slog.Logger
}

func (l *LogSink) Send(_ context.Context, m Message) error {
	l.Log.Info("mail (no SMTP configured, not delivered)",
		"to", m.To, "subject", m.Subject, "text", m.Text)
	return nil
}
