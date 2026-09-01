package infrastructure

import (
	"context"
	"net"
	"net/smtp"
	"testing"
	"time"
)

// TestSMTPEmailSender_send_RespectsContextCancellation proves an explicit
// ctx cancellation (not just ctx reaching its deadline) unblocks send()
// promptly, well before smtpSendTimeout — the exact gap code review found in
// the first version of the context.WithDeadline-only rewrite of this
// function (see docs/rfc/2026-09-02-backend-code-review-round2.md §3.5).
// The fake server here accepts the TCP connection but never writes the SMTP
// greeting, so send() would otherwise block inside smtp.NewClient reading
// that greeting until conn's deadline fires.
func TestSMTPEmailSender_send_RespectsContextCancellation(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Deliberately never write anything and never close — simulates a
		// hung SMTP server that accepted the connection but never responds.
		<-context.Background().Done()
		_ = conn
	}()

	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}

	sender := &SMTPEmailSender{
		host: host, port: port, from: "from@example.com",
		auth: smtp.PlainAuth("", "user", "pass", host),
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err = sender.send(ctx, "to@example.com", "subject", "body")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from a server that never responds")
	}
	// smtpSendTimeout is 10s — if cancellation weren't observed, this would
	// take that long (or the ctx.Deadline() computed at call start, which
	// there isn't one of here since ctx has no deadline, only Done()).
	// Generous upper bound to stay non-flaky under CI load while still
	// clearly proving cancellation was honored, not the 10s constant.
	if elapsed > 3*time.Second {
		t.Errorf("send() took %v to return after ctx was canceled at ~150ms — cancellation was not observed promptly", elapsed)
	}
}
