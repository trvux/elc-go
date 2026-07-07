package domain

import (
	"context"
	"io"
)

// Uploader stores a file under a folder prefix and returns its public URL.
// folder mirrors the old Supabase Storage folder-per-entity convention
// (e.g. "products", "categories", "branches") — implemented by infrastructure.
type Uploader interface {
	Upload(ctx context.Context, folder, filename, contentType string, content io.Reader) (url string, err error)
}
