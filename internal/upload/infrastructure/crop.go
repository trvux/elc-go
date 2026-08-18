package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"os/exec"
	"time"

	"github.com/kovidgoyal/imaging"

	"github.com/trvux/elc-go/internal/upload/domain"
)

// cropTargets are the additional aspect ratios Google's Merchant Listing
// guidance recommends alongside whatever ratio the original upload already
// is (this site standardizes product photos at 16:9) — see the "audit SEO
// trang chi tiết sản phẩm" work these came out of. w/h are a ceiling, not a
// forced size: fitRatio never upscales past the source image.
var cropTargets = map[string]struct{ w, h int }{
	"4:3": {1200, 900},
	"1:1": {1200, 1200},
}

// cwebpQuality mirrors the client's own WebP compression setting (see
// convertToWebP on the elc-tem side) so crop derivatives don't look
// noticeably worse than the primary gallery photo sitting next to them.
const cwebpQuality = 90

// fitRatio returns the largest w×h at the target's aspect ratio that still
// fits inside the source dimensions — imaging.Fill will upscale to hit
// whatever w×h you hand it, so callers that don't want to upscale a
// smaller-than-target source need to shrink the target first.
func fitRatio(srcW, srcH, targetW, targetH int) (int, int) {
	ratio := float64(targetW) / float64(targetH)
	w, h := targetW, targetH
	if w > srcW {
		w = srcW
		h = int(float64(w) / ratio)
	}
	if h > srcH {
		h = srcH
		w = int(float64(h) * ratio)
	}
	return w, h
}

// generateCropVariants center-crops decoded into cropTargets' ratios and
// converts each to WebP via cwebp (see encodeWebP) — imaging's pure-Go
// Encode has no WEBP case (only JPEG/PNG/GIF/TIFF/BMP), and no pure-Go
// lossy WebP encoder here has libwebp's track record, so this shells out to
// the real thing instead, keeping the Go binary itself cgo-free.
func generateCropVariants(decoded image.Image) (map[string][]byte, error) {
	srcW, srcH := decoded.Bounds().Dx(), decoded.Bounds().Dy()
	variants := make(map[string][]byte, len(cropTargets))
	for label, target := range cropTargets {
		w, h := fitRatio(srcW, srcH, target.w, target.h)
		cropped := imaging.Fill(decoded, w, h, imaging.Center, imaging.Lanczos)

		// PNG (lossless) intermediate — cwebp does the one and only lossy
		// pass. Piping a JPEG-encoded intermediate into cwebp would
		// compound two separate lossy generations onto the same image.
		var png bytes.Buffer
		if err := imaging.Encode(&png, cropped, imaging.PNG); err != nil {
			return nil, fmt.Errorf("encode %s crop (png intermediate): %w", label, err)
		}

		webp, err := encodeWebP(png.Bytes(), cwebpQuality)
		if err != nil {
			return nil, fmt.Errorf("encode %s crop (webp): %w", label, err)
		}
		variants[label] = webp
	}
	return variants, nil
}

// encodeWebP shells out to cwebp (libwebp's own CLI, apk package
// libwebp-tools) — "-- -" reads the source from stdin, "-o -" writes the
// encoded result to stdout, both confirmed to round-trip correctly against
// a real product photo before this shipped.
func encodeWebP(pngBytes []byte, quality int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cwebp", "-quiet", "-q", fmt.Sprintf("%d", quality), "-o", "-", "--", "-")
	cmd.Stdin = bytes.NewReader(pngBytes)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cwebp: %w (stderr: %s)", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// ImageCropper is the domain.ImageProcessor implementation backing the
// products-folder crop-on-upload feature.
type ImageCropper struct{}

var _ domain.ImageProcessor = (*ImageCropper)(nil)

func NewImageCropper() *ImageCropper { return &ImageCropper{} }

func (c *ImageCropper) GenerateCropVariants(content io.Reader) (map[string][]byte, error) {
	decoded, err := imaging.Decode(content)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return generateCropVariants(decoded)
}
