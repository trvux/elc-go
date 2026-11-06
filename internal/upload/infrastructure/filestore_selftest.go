package infrastructure

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/trvux/elc-go/internal/upload/domain"
)

// SelfTestFileStore proves the private store really works with the credentials it was given:
// write a probe object, read its size back, mint a signed link, delete it. Run once at
// start-up and logged, so a wrong key or a bucket without write access shows in the logs
// the day it is deployed, not the first time a customer's drawing is lost.
func SelfTestFileStore(ctx context.Context, store domain.FileStore) error {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	key := "healthcheck/" + hex.EncodeToString(random) + ".txt"
	payload := []byte("elc private file store self-test")

	if err := store.Put(ctx, key, "text/plain", bytes.NewReader(payload)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	// Whatever happens next, do not leave the probe behind.
	defer func() { _ = store.Delete(context.WithoutCancel(ctx), key) }()

	info, err := store.Stat(ctx, key)
	if err != nil {
		return fmt.Errorf("read back: %w", err)
	}
	if info.Size != int64(len(payload)) {
		return fmt.Errorf("read back: size %d, want %d", info.Size, len(payload))
	}
	if link, err := store.SignedURL(ctx, key, time.Minute, "probe.txt", "text/plain", false); err != nil || link == "" {
		return fmt.Errorf("sign: %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if _, err := store.Stat(ctx, key); err == nil {
		return fmt.Errorf("delete: object still there")
	}
	return nil
}
