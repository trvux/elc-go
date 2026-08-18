package infrastructure

import (
	"context"
	"fmt"
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
	host         string
	port         string
	username     string
	password     string
	from         string
	adminBaseURL string
	auth         smtp.Auth
}

var _ domain.EmailSender = (*SMTPEmailSender)(nil)

func NewSMTPEmailSender(host, port, username, password, from, adminBaseURL string) *SMTPEmailSender {
	return &SMTPEmailSender{
		host:         host,
		port:         port,
		username:     username,
		password:     password,
		from:         from,
		adminBaseURL: adminBaseURL,
		auth:         smtp.PlainAuth("", username, password, host),
	}
}

func (s *SMTPEmailSender) SendInvite(ctx context.Context, to string, rawToken string, role domain.Role) error {
	link := fmt.Sprintf("%s/admin/accept-invite?token=%s", s.adminBaseURL, rawToken)
	subject := "Bạn được mời quản trị dienmayelc.com.vn"
	body := fmt.Sprintf(
		"Bạn được mời làm %s trên hệ thống quản trị dienmayelc.com.vn.\n\n"+
			"Bấm vào đường dẫn sau để tạo tài khoản (hết hạn sau 72 giờ):\n%s\n\n"+
			"Nếu không phải bạn yêu cầu, hãy bỏ qua email này.",
		role, link,
	)
	return s.send(ctx, to, subject, body)
}

func (s *SMTPEmailSender) SendPasswordReset(ctx context.Context, to string, rawToken string) error {
	link := fmt.Sprintf("%s/admin/reset-password?token=%s", s.adminBaseURL, rawToken)
	subject := "Đặt lại mật khẩu quản trị dienmayelc.com.vn"
	body := fmt.Sprintf(
		"Có yêu cầu đặt lại mật khẩu cho tài khoản này.\n\n"+
			"Bấm vào đường dẫn sau để đặt mật khẩu mới (hết hạn sau 30 phút):\n%s\n\n"+
			"Nếu không phải bạn yêu cầu, hãy bỏ qua email này — mật khẩu hiện tại vẫn giữ nguyên.",
		link,
	)
	return s.send(ctx, to, subject, body)
}

// send bounds smtp.SendMail (which has no deadline of its own) to
// smtpSendTimeout, so a hung SMTP provider fails the request instead of
// blocking it indefinitely.
func (s *SMTPEmailSender) send(ctx context.Context, to, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)
	addr := s.host + ":" + s.port

	ctx, cancel := context.WithTimeout(ctx, smtpSendTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, s.auth, s.from, []string{to}, []byte(msg))
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("smtp send to %s: %w", to, ctx.Err())
	}
}

// LogEmailSender is the fallback used when SMTP isn't configured (local dev
// without real credentials). It logs the link instead of sending it, so
// development isn't blocked on having a mailbox set up.
type LogEmailSender struct {
	log          *zap.Logger
	adminBaseURL string
}

var _ domain.EmailSender = (*LogEmailSender)(nil)

func NewLogEmailSender(log *zap.Logger, adminBaseURL string) *LogEmailSender {
	return &LogEmailSender{log: log, adminBaseURL: adminBaseURL}
}

func (s *LogEmailSender) SendInvite(ctx context.Context, to string, rawToken string, role domain.Role) error {
	link := fmt.Sprintf("%s/admin/accept-invite?token=%s", s.adminBaseURL, rawToken)
	s.log.Warn("SMTP not configured — logging invite link instead of emailing it",
		zap.String("to", to), zap.String("role", string(role)), zap.String("link", link))
	return nil
}

func (s *LogEmailSender) SendPasswordReset(ctx context.Context, to string, rawToken string) error {
	link := fmt.Sprintf("%s/admin/reset-password?token=%s", s.adminBaseURL, rawToken)
	s.log.Warn("SMTP not configured — logging password reset link instead of emailing it",
		zap.String("to", to), zap.String("link", link))
	return nil
}
