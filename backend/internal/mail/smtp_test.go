package mail

// The SMTP client against a scripted relay. Mailpit covers the plaintext
// conversation every day in dev; what it never exercises is exactly what a
// real relay demands — STARTTLS, implicit TLS, AUTH, a deadline — so those
// paths are proven here, on 127.0.0.1, with a self-signed authority the
// client is told to trust through its unexported test hooks.

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRelay is just enough of RFC 5321 to answer one client session the way
// a real relay would, recording what it was told. Each connection is a tiny
// state machine driven by the verb at the start of every line.
type fakeRelay struct {
	ln       net.Listener
	tlsConf  *tls.Config // nil = TLS never offered
	implicit bool        // TLS from the first byte (the port-465 shape)
	wantAuth string      // the AUTH PLAIN blob the relay accepts; "" = no AUTH offered
	reject   bool        // answer the message body with 554 instead of 250
	silent   bool        // accept the connection and never speak (a stalled relay)

	mu       sync.Mutex
	commands []string
	from, to string
	data     string
	tlsUsed  bool
	authed   bool
}

func (f *fakeRelay) record(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

func (f *fakeRelay) addr() string { return f.ln.Addr().String() }

func (f *fakeRelay) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	if f.silent {
		<-time.After(5 * time.Second) // longer than any timeout under test
		return
	}
	if f.implicit {
		conn = tls.Server(conn, f.tlsConf)
		f.record(func() { f.tlsUsed = true })
	}
	r := bufio.NewReader(conn)
	reply := func(s string) { _, _ = fmt.Fprint(conn, s+"\r\n") }
	reply("220 fake relay")
	tlsActive := f.implicit
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		f.record(func() { f.commands = append(f.commands, line) })
		verb := strings.ToUpper(strings.SplitN(line, " ", 2)[0])
		switch verb {
		case "EHLO", "HELO":
			// A multi-line reply: "250-" continues, "250 " (space) ends it.
			lines := []string{"250-fake"}
			if f.tlsConf != nil && !tlsActive {
				lines = append(lines, "250-STARTTLS")
			}
			if f.wantAuth != "" {
				lines = append(lines, "250-AUTH PLAIN")
			}
			reply(strings.Join(append(lines, "250 OK"), "\r\n"))
		case "STARTTLS":
			reply("220 go ahead")
			tc := tls.Server(conn, f.tlsConf)
			if err := tc.Handshake(); err != nil {
				return
			}
			conn, r, tlsActive = tc, bufio.NewReader(tc), true
			f.record(func() { f.tlsUsed = true })
		case "AUTH":
			if strings.TrimPrefix(line, "AUTH PLAIN ") == f.wantAuth {
				f.record(func() { f.authed = true })
				reply("235 authenticated")
			} else {
				reply("535 bad credentials")
			}
		case "MAIL":
			f.record(func() { f.from = line })
			reply("250 sender ok")
		case "RCPT":
			f.record(func() { f.to = line })
			reply("250 recipient ok")
		case "DATA":
			reply("354 go ahead")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				b.WriteString(l)
			}
			f.record(func() { f.data = b.String() })
			if f.reject {
				reply("554 message refused")
			} else {
				reply("250 queued")
			}
		case "QUIT":
			reply("221 bye")
			return
		default:
			reply("500 unknown command")
		}
	}
}

// newRelay starts the scripted relay on an ephemeral loopback port and
// returns it together with the CA pool the client must trust.
func newRelay(t *testing.T, f *fakeRelay) (*fakeRelay, *x509.CertPool) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f.ln = ln
	t.Cleanup(func() { _ = ln.Close() })

	var pool *x509.CertPool
	if f.tlsConf != nil || f.implicit {
		cert, p := selfSigned(t)
		f.tlsConf = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
		pool = p
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed by Cleanup
			}
			go f.serve(conn)
		}
	}()
	return f, pool
}

// selfSigned mints a throwaway certificate authority valid for 127.0.0.1.
// The client verifies the relay's certificate against this pool exactly as
// it would verify Resend's against the system store.
func selfSigned(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "fake relay"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, pool
}

func plainBlob(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte("\x00" + user + "\x00" + pass))
}

var testMessage = Message{
	To:      "customer@example.test",
	Subject: "Վերականգնել գաղտնաբառը", // Armenian: the subject that once shipped as mojibake
	Text:    "Բարև — hello\n",
}

func sawCommand(f *fakeRelay, prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.commands {
		if strings.HasPrefix(strings.ToUpper(c), prefix) {
			return true
		}
	}
	return false
}

func TestSMTP_PlaintextRelayLikeMailpit(t *testing.T) {
	relay, _ := newRelay(t, &fakeRelay{})
	s := &SMTP{Addr: relay.addr(), From: "Mountain Breath <hive@mountainbreath.test>"}

	if err := s.Send(context.Background(), testMessage); err != nil {
		t.Fatalf("Send: %v", err)
	}
	relay.mu.Lock()
	defer relay.mu.Unlock()
	if relay.from != "MAIL FROM:<hive@mountainbreath.test>" {
		t.Errorf("envelope sender = %q, want the bare address without the display name", relay.from)
	}
	if relay.to != "RCPT TO:<customer@example.test>" {
		t.Errorf("envelope recipient = %q", relay.to)
	}
	for _, want := range []string{
		"From: Mountain Breath <hive@mountainbreath.test>\r\n",
		"Subject: =?utf-8?q?", // RFC 2047: the Armenian subject is encoded, never raw
		"Content-Type: text/plain; charset=utf-8\r\n",
		"\r\n\r\nԲարև — hello",
	} {
		if !strings.Contains(relay.data, want) {
			t.Errorf("message lacks %q:\n%s", want, relay.data)
		}
	}
	if relay.tlsUsed || relay.authed {
		t.Errorf("a relay offering neither TLS nor AUTH got tls=%v auth=%v", relay.tlsUsed, relay.authed)
	}
}

func TestSMTP_RefusesCredentialsInClear(t *testing.T) {
	relay, _ := newRelay(t, &fakeRelay{wantAuth: plainBlob("resend", "re_secret")})
	s := &SMTP{Addr: relay.addr(), From: "hive@mountainbreath.test", Username: "resend", Password: "re_secret"}

	err := s.Send(context.Background(), testMessage)
	if err == nil || !strings.Contains(err.Error(), "refusing to send credentials in clear") {
		t.Fatalf("err = %v, want the plaintext refusal", err)
	}
	// The refusal must come BEFORE anything secret or substantive is sent.
	if sawCommand(relay, "AUTH") || sawCommand(relay, "MAIL FROM") {
		t.Errorf("client kept talking after the refusal: %v", relay.commands)
	}
}

func TestSMTP_STARTTLSThenAuth(t *testing.T) {
	relay, pool := newRelay(t, &fakeRelay{tlsConf: &tls.Config{}, wantAuth: plainBlob("resend", "re_secret")})
	s := &SMTP{
		Addr: relay.addr(), From: "hive@mountainbreath.test",
		Username: "resend", Password: "re_secret",
		rootCAs: pool,
	}

	if err := s.Send(context.Background(), testMessage); err != nil {
		t.Fatalf("Send: %v", err)
	}
	relay.mu.Lock()
	defer relay.mu.Unlock()
	if !relay.tlsUsed {
		t.Error("relay offered STARTTLS and the client did not upgrade")
	}
	if !relay.authed {
		t.Error("client never authenticated")
	}
	if !strings.Contains(relay.data, "Subject: =?utf-8?q?") {
		t.Errorf("message not delivered over the upgraded connection:\n%s", relay.data)
	}
}

func TestSMTP_ImplicitTLS(t *testing.T) {
	relay, pool := newRelay(t, &fakeRelay{implicit: true, wantAuth: plainBlob("resend", "re_secret")})
	s := &SMTP{
		Addr: relay.addr(), From: "hive@mountainbreath.test",
		Username: "resend", Password: "re_secret",
		rootCAs: pool, implicitTLS: true, // the port-465 path, on an ephemeral port
	}

	if err := s.Send(context.Background(), testMessage); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sawCommand(relay, "STARTTLS") {
		t.Error("STARTTLS sent on a connection that was TLS from the first byte")
	}
	relay.mu.Lock()
	defer relay.mu.Unlock()
	if !relay.tlsUsed || !relay.authed || relay.data == "" {
		t.Errorf("tls=%v auth=%v delivered=%v", relay.tlsUsed, relay.authed, relay.data != "")
	}
}

func TestSMTP_WrongPasswordIsAnError(t *testing.T) {
	relay, pool := newRelay(t, &fakeRelay{tlsConf: &tls.Config{}, wantAuth: plainBlob("resend", "right")})
	s := &SMTP{Addr: relay.addr(), From: "hive@mountainbreath.test", Username: "resend", Password: "wrong", rootCAs: pool}

	err := s.Send(context.Background(), testMessage)
	if err == nil || !strings.Contains(err.Error(), "AUTH") || !strings.Contains(err.Error(), "535") {
		t.Fatalf("err = %v, want the relay's 535 surfaced as an AUTH error", err)
	}
}

func TestSMTP_RelayRefusingTheMessageIsAnError(t *testing.T) {
	relay, _ := newRelay(t, &fakeRelay{reject: true})
	s := &SMTP{Addr: relay.addr(), From: "hive@mountainbreath.test"}

	err := s.Send(context.Background(), testMessage)
	if err == nil || !strings.Contains(err.Error(), "554") {
		t.Fatalf("err = %v, want the 554 verdict on the body surfaced", err)
	}
}

func TestSMTP_StalledRelayHitsTheDeadline(t *testing.T) {
	relay, _ := newRelay(t, &fakeRelay{silent: true})
	s := &SMTP{Addr: relay.addr(), From: "hive@mountainbreath.test", Timeout: 200 * time.Millisecond}

	start := time.Now()
	err := s.Send(context.Background(), testMessage)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Send returned nil against a relay that never spoke")
	}
	if elapsed > 2*time.Second {
		t.Errorf("Send took %v against a stalled relay; the 200ms deadline did not bite", elapsed)
	}
}

func TestSMTP_CancelledContextIsHonoured(t *testing.T) {
	relay, _ := newRelay(t, &fakeRelay{silent: true})
	s := &SMTP{Addr: relay.addr(), From: "hive@mountainbreath.test"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.Send(ctx, testMessage); err == nil {
		t.Fatal("Send returned nil on an already-cancelled context")
	}
}

func TestSMTP_AddressNeedsAPort(t *testing.T) {
	s := &SMTP{Addr: "smtp.resend.com", From: "hive@mountainbreath.test"}
	err := s.Send(context.Background(), testMessage)
	if err == nil || !strings.Contains(err.Error(), "host:port") {
		t.Fatalf("err = %v, want a host:port complaint before any dial", err)
	}
}

func TestWantsImplicitTLS(t *testing.T) {
	for port, want := range map[string]bool{"465": true, "2465": true, "587": false, "25": false, "1025": false} {
		if got := wantsImplicitTLS(port); got != want {
			t.Errorf("wantsImplicitTLS(%q) = %v, want %v", port, got, want)
		}
	}
}
