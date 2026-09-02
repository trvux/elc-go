package infrastructure

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"go.uber.org/zap"

	"github.com/trvux/elc-go/internal/auth/domain"
)

// smtpSendTimeout bounds how long a single SMTP send may take. net/smtp
// has no built-in deadline, so a hung/slow provider would otherwise block
// the request (and its goroutine/DB connection) until the OS TCP timeout.
const smtpSendTimeout = 10 * time.Second

// SMTPEmailSender sends real emails over SMTP — works with any provider that
// speaks SMTP (Gmail app password, Amazon SES, Resend, SendGrid, ...), so
// swapping providers later is a config change, not a code change.
type SMTPEmailSender struct {
	host       string
	port       string
	username   string
	password   string
	from       string
	appBaseURL string
	auth       smtp.Auth
}

var _ domain.EmailSender = (*SMTPEmailSender)(nil)

func NewSMTPEmailSender(host, port, username, password, from, appBaseURL string) *SMTPEmailSender {
	return &SMTPEmailSender{
		host:       host,
		port:       port,
		username:   username,
		password:   password,
		from:       from,
		appBaseURL: appBaseURL,
		auth:       smtp.PlainAuth("", username, password, host),
	}
}

// SendMagicLink emails both redemption paths for the same token: a clickable
// link carrying the raw token in a URL fragment (never a query param — see
// application.RequestMagicLink) and the 6-digit code for manual entry.
func (s *SMTPEmailSender) SendMagicLink(ctx context.Context, to string, rawToken string, code string) error {
	link := fmt.Sprintf("%s/magic-link#%s", s.appBaseURL, rawToken)
	subject := "Đăng nhập dienmayelc.com.vn"
	body := fmt.Sprintf(
		"Bấm vào đường dẫn sau để đăng nhập (hết hạn sau 10 phút):\n%s\n\n"+
			"Hoặc nhập mã: %s\n\n"+
			"Nếu không phải bạn yêu cầu, hãy bỏ qua email này.",
		link, code,
	)
	return s.send(ctx, to, subject, body)
}

// send is a context-aware reimplementation of smtp.SendMail's steps
// (dial -> optional STARTTLS -> auth -> mail/rcpt/data -> quit), because
// smtp.SendMail itself accepts no context and has no way to bound how long
// it blocks. The previous version wrapped smtp.SendMail in a goroutine +
// select on ctx.Done(), but that only made the *caller* stop waiting — the
// goroutine (and its underlying TCP connection) kept running in the
// background until the OS eventually timed it out on its own, sometimes
// minutes later. Here, DialContext bounds the connection step directly, and
// conn.SetDeadline bounds every read/write of the SMTP conversation that
// follows: once the deadline passes, whatever blocking call is in flight on
// this same goroutine fails immediately with an i/o timeout — no separate
// goroutine ever exists to leak. See
// docs/rfc/2026-09-02-backend-code-review-round2.md §3.5.
func (s *SMTPEmailSender) send(ctx context.Context, to, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)
	addr := s.host + ":" + s.port

	deadline := time.Now().Add(smtpSendTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	dialCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("smtp set deadline: %w", err)
	}

	// conn.SetDeadline above bounds the conversation by wall-clock time, but
	// an explicit ctx cancellation (as opposed to ctx merely reaching its
	// deadline — e.g. the caller's own request context canceled because the
	// client disconnected) needs its own watch: it can fire earlier than
	// `deadline` and nothing else here re-checks ctx once the TCP connection
	// is open. This goroutine's lifetime is bounded by construction — it
	// always exits via whichever of ctx.Done()/done fires first, and done is
	// guaranteed to close (via the defer below) by the time this function
	// returns — so, unlike the old goroutine+select design this function
	// itself replaced, it can never outlive the call and cannot leak.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.SetDeadline(time.Now())
		case <-done:
		}
	}()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if err := client.Auth(s.auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close body: %w", err)
	}
	return client.Quit()
}

// LogEmailSender is the fallback used when SMTP isn't configured (local dev
// without real credentials). It logs the link instead of sending it, so
// development isn't blocked on having a mailbox set up.
type LogEmailSender struct {
	log        *zap.Logger
	appBaseURL string
}

var _ domain.EmailSender = (*LogEmailSender)(nil)

func NewLogEmailSender(log *zap.Logger, appBaseURL string) *LogEmailSender {
	return &LogEmailSender{log: log, appBaseURL: appBaseURL}
}

func (s *LogEmailSender) SendMagicLink(ctx context.Context, to string, rawToken string, code string) error {
	link := fmt.Sprintf("%s/magic-link#%s", s.appBaseURL, rawToken)
	s.log.Warn("SMTP not configured — logging magic link instead of emailing it",
		zap.String("to", to), zap.String("code", code), zap.String("link", link))
	return nil
}
