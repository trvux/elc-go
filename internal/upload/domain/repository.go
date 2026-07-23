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

// ImageProcessor generates derivative crops of an uploaded image — kept
// separate from Uploader (which only knows how to store already-encoded
// bytes) since crop generation is pure compute, no storage concern. Returns
// a label ("4:3", "1:1") -> encoded WebP bytes map; see
// infrastructure.ImageCropper for the concrete implementation.
type ImageProcessor interface {
	GenerateCropVariants(content io.Reader) (map[string][]byte, error)
}
