package presentation

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/upload/domain"
)

// maxUploadBytes caps the request body read — the client already downsizes
// to WebP before sending, so anything past this is either a mistake or abuse.
const maxUploadBytes = 10 << 20 // 10 MB

type UploadHandler struct {
	uploader domain.Uploader
}

func NewUploadHandler(uploader domain.Uploader) *UploadHandler {
	return &UploadHandler{uploader: uploader}
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("file too large or invalid multipart body", nil))
		return
	}

	folder := strings.Trim(r.FormValue("folder"), "/")
	if folder == "" {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"folder": {"folder is required"},
		}))
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

	filename := generateFilename(contentType)

	url, err := h.uploader.Upload(r.Context(), folder, filename, contentType, file)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, uploadResponse{URL: url})
}

// generateFilename mirrors the old Supabase client's `${Date.now()}-${random}.ext`
// naming so object keys stay readable/sortable in the bucket browser.
func generateFilename(contentType string) string {
	ext := "webp"
	if parts := strings.SplitN(contentType, "/", 2); len(parts) == 2 && parts[1] != "" {
		ext = parts[1]
	}

	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes)

	return time.Now().UTC().Format("20060102150405") + "-" + hex.EncodeToString(randBytes) + "." + ext
}
