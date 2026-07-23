package presentation

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/upload/domain"
)

// maxUploadBytes caps the request body read — the client already downsizes
// to WebP before sending, so anything past this is either a mistake or abuse.
const maxUploadBytes = 10 << 20 // 10 MB

// cropVariantsFolder is the only folder that gets extra aspect-ratio crops
// generated — product photos are what Product JSON-LD's image[] serves to
// Google's Merchant Listing rich result; nothing else (branch/category/news
// photos, avatars, ...) needs them, so every other folder keeps the
// original one-file-in-one-URL-out behavior unchanged.
const cropVariantsFolder = "products"

type UploadHandler struct {
	uploader  domain.Uploader
	processor domain.ImageProcessor
}

func NewUploadHandler(uploader domain.Uploader, processor domain.ImageProcessor) *UploadHandler {
	return &UploadHandler{uploader: uploader, processor: processor}
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

	var cropVariants map[string]string
	if folder == cropVariantsFolder {
		cropVariants = h.uploadCropVariants(r.Context(), file, filename)
		// Crop generation reads the file to decode it — rewind before the
		// original upload below reads it again from the start.
		if _, err := file.Seek(0, 0); err != nil {
			httpserver.WriteError(w, apperr.NewInternalError(err))
			return
		}
	}

	url, err := h.uploader.Upload(r.Context(), folder, filename, contentType, file)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, uploadResponse{URL: url, CropVariants: cropVariants})
}

// uploadCropVariants best-efforts the extra aspect-ratio crops — a decode
// failure (corrupt/unsupported source image) or a crop upload failure logs
// and returns nil rather than failing the whole request, since the primary
// image (uploaded by the caller right after this returns) is what the
// product actually needs; the crops only enrich JSON-LD.
func (h *UploadHandler) uploadCropVariants(ctx context.Context, file multipart.File, baseFilename string) map[string]string {
	crops, err := h.processor.GenerateCropVariants(file)
	if err != nil {
		log.Printf("upload: crop variants: generate: %v", err)
		return nil
	}

	base := strings.TrimSuffix(baseFilename, filepath.Ext(baseFilename))
	urls := make(map[string]string, len(crops))
	for label, bytesContent := range crops {
		cropFilename := fmt.Sprintf("%s-%s.webp", base, strings.ReplaceAll(label, ":", "x"))
		url, err := h.uploader.Upload(ctx, cropVariantsFolder, cropFilename, "image/webp", bytes.NewReader(bytesContent))
		if err != nil {
			log.Printf("upload: crop variants: upload %s: %v", label, err)
			continue
		}
		urls[label] = url
	}
	if len(urls) == 0 {
		return nil
	}
	return urls
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
