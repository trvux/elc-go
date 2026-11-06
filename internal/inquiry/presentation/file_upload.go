package presentation

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/trvux/elc-go/internal/inquiry/application"
	"github.com/trvux/elc-go/internal/inquiry/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	uploaddomain "github.com/trvux/elc-go/internal/upload/domain"
)

const (
	// multipartOverhead is allowed on top of the file itself for boundaries and headers.
	multipartOverhead = 1 << 20
	// maxConcurrentFileUploads bounds memory: each upload is held whole (<= 15 MB) so its
	// type can be checked and the store given a seekable body. Four at once is 60 MB.
	maxConcurrentFileUploads = 4
	// signedURLTTL is how long a staff member's link to a customer file works.
	signedURLTTL = 5 * time.Minute
)

type uploadedFileResponse struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}

// UploadFile receives one customer file (a drawing, a photo) before the inquiry exists and
// stores it in the PRIVATE bucket. Public and unauthenticated like Create, so it trusts
// nothing the client says: the type comes from the file's own first bytes, the extension
// and key from that type, and the size from what was actually read.
func (h *InquiryHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if !h.fileLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}
	select {
	case h.fileSlots <- struct{}{}:
		defer func() { <-h.fileSlots }()
	default:
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("the server is busy receiving other files, please try again in a moment"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxAttachmentBytes+multipartOverhead)
	reader, err := r.MultipartReader()
	if err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid multipart body", nil))
		return
	}

	var (
		data []byte
		name string
	)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			httpserver.WriteError(w, apperr.NewValidationError("file too large or invalid multipart body", map[string][]string{
				"file": {"file must be 15 MB or smaller"},
			}))
			return
		}
		if part.FormName() != "file" {
			continue
		}
		name = part.FileName()
		// One byte past the cap is read so "exactly 15 MB" and "too big" can be told apart.
		data, err = io.ReadAll(io.LimitReader(part, domain.MaxAttachmentBytes+1))
		if err != nil {
			httpserver.WriteError(w, apperr.NewValidationError("could not read the file", nil))
			return
		}
		break
	}

	if len(data) == 0 {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{"file": {"file is required"}}))
		return
	}
	if len(data) > domain.MaxAttachmentBytes {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{"file": {"file must be 15 MB or smaller"}}))
		return
	}

	head := data
	if len(head) > 2048 {
		head = head[:2048]
	}
	contentType, ext, ok := domain.DetectFileType(head)
	if !ok {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"file": {"only PDF, DWG, DXF, JPG, PNG and WebP files are accepted"},
		}))
		return
	}

	key, err := newAttachmentKey(ext)
	if err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}
	if err := h.files.Put(r.Context(), key, contentType, bytes.NewReader(data)); err != nil {
		h.log.Error("customer file upload failed", zap.Error(err))
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	displayName := domain.SanitizeAttachmentName(name)
	if displayName == "" {
		displayName = "tep-dinh-kem." + ext
	}
	httpserver.WriteJSON(w, http.StatusOK, uploadedFileResponse{Key: key, Name: displayName, Size: int64(len(data)), Type: contentType})
}

// newAttachmentKey: inquiries/<yyyymm>/<128 random bits>.<ext>. The month folder is for
// whoever browses the bucket; the random part is what makes a key impossible to guess.
func newAttachmentKey(ext string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return "inquiries/" + time.Now().UTC().Format("200601") + "/" + hex.EncodeToString(randomBytes) + "." + ext, nil
}

// verifyAttachments checks every private file a lead refers to: it must exist in the store,
// and its size and type are replaced by what the store reports, so a lead can neither claim
// a file it never uploaded nor understate its size. Legacy public URLs pass through.
func (h *InquiryHandler) verifyAttachments(r *http.Request, attachments []domain.Attachment) ([]domain.Attachment, error) {
	verified := make([]domain.Attachment, len(attachments))
	for i, a := range attachments {
		if !a.IsPrivate() {
			verified[i] = a
			continue
		}
		if !domain.ValidAttachmentKey(a.Key) {
			return nil, apperr.NewValidationError("validation failed", map[string][]string{"attachments": {"attachment key is not valid"}})
		}
		info, err := h.files.Stat(r.Context(), a.Key)
		if errors.Is(err, uploaddomain.ErrFileNotFound) {
			return nil, apperr.NewValidationError("validation failed", map[string][]string{"attachments": {"a file was not found, please upload it again"}})
		}
		if err != nil {
			h.log.Error("customer file check failed", zap.Error(err))
			return nil, apperr.NewInternalError(err)
		}
		a.Size, a.Type = info.Size, info.ContentType
		verified[i] = a
	}
	return verified, nil
}

type attachmentLinkResponse struct {
	URL       string `json:"url"`
	Name      string `json:"name"`
	Size      int64  `json:"size,omitempty"`
	Type      string `json:"type,omitempty"`
	ExpiresIn int    `json:"expires_in"`
}

// AttachmentURL gives a signed-in staff member a short-lived link to one file of a lead, by
// its position in the lead's attachments. 404 means the file is gone (the bucket drops
// customer files after 12 months), which the admin shows as "đã hết hạn".
func (h *InquiryHandler) AttachmentURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	index, err := strconv.Atoi(chi.URLParam(r, "index"))
	if err != nil || index < 0 {
		httpserver.WriteError(w, apperr.NewValidationError("invalid attachment index", nil))
		return
	}

	inquiry, err := application.GetInquiryByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if inquiry == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("inquiry"))
		return
	}
	attachments := inquiry.Attachments()
	if index >= len(attachments) {
		httpserver.WriteError(w, apperr.NewNotFoundError("attachment"))
		return
	}

	a := attachments[index]
	if !a.IsPrivate() {
		httpserver.WriteJSON(w, http.StatusOK, attachmentLinkResponse{URL: a.URL, Name: a.URL[lastSlash(a.URL)+1:]})
		return
	}
	if _, err := h.files.Stat(r.Context(), a.Key); err != nil {
		if errors.Is(err, uploaddomain.ErrFileNotFound) {
			httpserver.WriteError(w, apperr.NewNotFoundError("attachment"))
			return
		}
		h.log.Error("customer file check failed", zap.Error(err))
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	inline := a.Type == domain.TypePDF || a.Type == domain.TypeJPEG || a.Type == domain.TypePNG || a.Type == domain.TypeWebP
	link, err := h.files.SignedURL(r.Context(), a.Key, signedURLTTL, a.Name, a.Type, inline)
	if err != nil {
		h.log.Error("customer file signing failed", zap.Error(err))
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}
	userID, _ := httpserver.UserIDFromContext(r.Context())
	h.log.Info("customer file link issued", zap.String("user_id", userID), zap.String("inquiry_id", id), zap.Int("index", index))
	httpserver.WriteJSON(w, http.StatusOK, attachmentLinkResponse{URL: link, Name: a.Name, Size: a.Size, Type: a.Type, ExpiresIn: int(signedURLTTL.Seconds())})
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}
