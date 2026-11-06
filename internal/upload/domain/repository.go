package domain

import (
	"context"
	"errors"
	"io"
	"time"
)

// Uploader stores a file under a folder prefix and returns its public URL.
// folder mirrors the old Supabase Storage folder-per-entity convention
// (e.g. "products", "categories", "branches") — implemented by infrastructure.
type Uploader interface {
	Upload(ctx context.Context, folder, filename, contentType string, content io.Reader) (url string, err error)
}

// ImageProcessor generates derivative crops of an uploaded image — kept
// separate from Uploader (which only knows how to store already-encoded
// bytes) since crop generation is pure compute, no storage concern. Returns
// a label ("4:3", "1:1") -> encoded WebP bytes map; see
// infrastructure.ImageCropper for the concrete implementation.
type ImageProcessor interface {
	GenerateCropVariants(content io.Reader) (map[string][]byte, error)
}

// FileInfo is what a private store knows about an object it holds.
type FileInfo struct {
	Size        int64
	ContentType string
}

// ErrFileNotFound is returned by FileStore.Stat when the object does not exist (never
// uploaded, or removed by the bucket's retention rule).
var ErrFileNotFound = errors.New("file not found")

// FileStore holds customer files that must NOT be publicly readable (drawings, photos of
// a site): unlike Uploader there is no public URL, access is a short-lived signed link
// minted for a signed-in staff member.
type FileStore interface {
	Put(ctx context.Context, key, contentType string, body io.ReadSeeker) error
	Stat(ctx context.Context, key string) (FileInfo, error)
	// SignedURL returns a link that works for ttl. name is the file name offered on download;
	// inline asks the browser to display the file (images, PDF) instead of saving it.
	SignedURL(ctx context.Context, key string, ttl time.Duration, name, contentType string, inline bool) (string, error)
	Delete(ctx context.Context, key string) error
}
