// Package mail is the one place the application sends email from.
//
// The interface is deliberately one method, declared here at the point of
// use (the api layer consumes it, exactly like its store interfaces): a
// handler needs "deliver this message" and has no business knowing whether
// that means an SMTP conversation, a provider's REST API, or a test's slice
// of sent messages. Production (decision #104) is Resend's relay, spoken to
// over SMTP — the very conversation dev has with Mailpit — so the first real
// relay exercised exactly the code the dev catcher did.
package mail

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
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

// DefaultTimeout bounds one SMTP conversation — dial, TLS handshake, AUTH,
// the DATA transfer, QUIT — when SMTP.Timeout is zero. Every handler sends
// synchronously before answering the browser, so without a clock a stalled
// relay across the internet would hold a checkout response open with no
// limit at all. Mail was already non-fatal; this gives it a deadline.
const DefaultTimeout = 10 * time.Second

// SMTP delivers through a relay — Mailpit's catcher in dev (localhost:1025,
// browse the mailbox at :8025), Resend's relay in prod.
//
// Transport security follows the port, the way every mail client decides
// it: port 465 means implicit TLS (the connection IS a TLS connection from
// the first byte); any other port starts in plaintext and upgrades with
// STARTTLS when the server offers it. Credentials never travel over a
// plaintext connection — a relay that wants a password but offers no TLS
// is refused, not accommodated.
type SMTP struct {
	Addr     string // host:port
	From     string // "Mountain Breath <hive@example.com>"
	Username string // empty = no AUTH (the dev catcher has none)
	Password string
	Timeout  time.Duration // zero = DefaultTimeout

	// Test hooks, unexported: the TLS paths are exactly what Mailpit never
	// exercises, so the tests run them against a scripted relay on an
	// ephemeral port with a self-signed authority. Production leaves both
	// zero — the system trust store, and the port rule above.
	rootCAs     *x509.CertPool
	implicitTLS bool
}

// wantsImplicitTLS is the port rule: 465 (and its alternate 2465) are the
// SMTPS ports, TLS from the first byte. Everything else negotiates.
func wantsImplicitTLS(port string) bool {
	return port == "465" || port == "2465"
}

// Send speaks just enough SMTP. Two encoding details matter, because SMTP
// predates UTF-8 by a decade:
//
//   - the Subject must be RFC 2047-encoded or an Armenian "Վերականգնել…"
//     arrives as mojibake — mime.QEncoding does the =?utf-8?q?…?= dance;
//   - the body carries an explicit text/plain; charset=utf-8 header, since
//     the assumed default is US-ASCII.
//
// net/smtp's convenience smtp.SendMail does the same seven steps this does,
// but takes no context, sets no deadline, and cannot do implicit TLS — so
// this drives the underlying smtp.Client by hand, on a connection that
// carries a deadline. Dropping one level below the convenience wrapper is
// the whole change; the protocol conversation is the standard library's.
func (s *SMTP) Send(ctx context.Context, m Message) error {
	host, port, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return fmt.Errorf("mail relay address %q: want host:port: %w", s.Addr, err)
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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

	// One deadline for the whole conversation. A net.Conn deadline is an
	// absolute instant that governs every later Read and Write, so the TLS
	// handshake, AUTH and the body transfer all inherit it with no
	// per-step plumbing — the C++ analogue is a single steady_clock
	// time_point checked by every blocking call, rather than a duration
	// passed down through each function.
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("connecting to mail relay %s: %w", s.Addr, err)
	}
	defer func() { _ = conn.Close() }() // a no-op after a clean QUIT; the safety net on every error path
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("mail relay %s: setting deadline: %w", s.Addr, err)
	}

	tlsConfig := &tls.Config{ServerName: host, RootCAs: s.rootCAs, MinVersion: tls.VersionTLS12}
	encrypted := false
	if wantsImplicitTLS(port) || s.implicitTLS {
		conn = tls.Client(conn, tlsConfig) // the handshake happens on the first read, under the deadline
		encrypted = true
	}

	c, err := smtp.NewClient(conn, host) // reads the 220 greeting
	if err != nil {
		return fmt.Errorf("mail relay %s: greeting: %w", s.Addr, err)
	}
	defer func() { _ = c.Close() }()

	if !encrypted {
		// Extension sends EHLO and reads the server's capability list.
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("mail relay %s: STARTTLS: %w", s.Addr, err)
			}
			encrypted = true
		}
	}
	if s.Username != "" {
		// net/smtp's PlainAuth makes this check too, but exempts localhost;
		// no exemption here — the rule is the same on every address, which
		// is also what lets the tests prove it on 127.0.0.1.
		if !encrypted {
			return fmt.Errorf("mail relay %s offers no TLS: refusing to send credentials in clear", s.Addr)
		}
		if err := c.Auth(smtp.PlainAuth("", s.Username, s.Password, host)); err != nil {
			return fmt.Errorf("mail relay %s: AUTH: %w", s.Addr, err)
		}
	}

	if err := c.Mail(fromAddress(s.From)); err != nil {
		return fmt.Errorf("mail relay %s: MAIL FROM: %w", s.Addr, err)
	}
	if err := c.Rcpt(m.To); err != nil {
		return fmt.Errorf("mail relay %s: RCPT TO %s: %w", s.Addr, m.To, err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mail relay %s: DATA: %w", s.Addr, err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		return fmt.Errorf("sending mail to %s: %w", m.To, err)
	}
	// Close sends the end-of-data marker and reads the relay's verdict on
	// the whole message — "250 queued" or a rejection — so THIS is the
	// error that means "not delivered", not the Write above.
	if err := w.Close(); err != nil {
		return fmt.Errorf("sending mail to %s: relay refused the message: %w", m.To, err)
	}
	if err := c.Quit(); err != nil {
		return fmt.Errorf("mail relay %s: QUIT: %w", s.Addr, err)
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
