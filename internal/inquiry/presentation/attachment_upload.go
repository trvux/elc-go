package presentation

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// maxAttachmentBytes mirrors internal/upload's cap — the client downsizes
// to a reasonable size before sending, anything past this is a mistake or abuse.
const maxAttachmentBytes = 10 << 20 // 10 MB

// attachmentFolder is hardcoded — unlike the admin uploader
// (internal/upload), this endpoint is unauthenticated, so it must never
// accept a client-supplied folder.
const attachmentFolder = "inquiries"

type attachmentUploadResponse struct {
	URL string `json:"url"`
}

// UploadAttachment lets an anonymous visitor attach photos to a lead before
// the inquiry itself exists (e.g. a photo of a leaking AC unit for a repair
// request, or the job site for a project request) — public, unauthenticated,
// same anonymous-submission posture as Create, but with its own more
// generous rate limit since one visitor may attach several photos in one
// session. No crop-variant generation here — that's product-photo-specific,
// see internal/upload/presentation.
func (h *InquiryHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	if !h.uploadLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentBytes)
	if err := r.ParseMultipartForm(maxAttachmentBytes); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("file too large or invalid multipart body", nil))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"file": {"file is required"},
		}))
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" || !strings.HasPrefix(contentType, "image/") {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"file": {"only image uploads are allowed"},
		}))
		return
	}

	filename, err := generateAttachmentFilename(contentType)
	if err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	url, err := h.uploader.Upload(r.Context(), attachmentFolder, filename, contentType, file)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, attachmentUploadResponse{URL: url})
}

// generateAttachmentFilename mirrors internal/upload/presentation's
// generateFilename — duplicated rather than shared across the module
// boundary, consistent with this codebase's DDD module separation (business
// modules depend on each other's domain interfaces, not presentation internals).
func generateAttachmentFilename(contentType string) (string, error) {
	ext := "webp"
	if parts := strings.SplitN(contentType, "/", 2); len(parts) == 2 && parts[1] != "" {
		ext = parts[1]
	}

	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("generate attachment filename: %w", err)
	}

	return time.Now().UTC().Format("20060102150405") + "-" + hex.EncodeToString(randBytes) + "." + ext, nil
}
