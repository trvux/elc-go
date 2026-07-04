package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

type errorResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Fields  map[string][]string `json:"fields,omitempty"`
}

// logger is set once at startup via SetLogger. It exists only so WriteError
// can log the real error behind a 5xx before responding — every other part
// of the app receives its logger via explicit injection, not this global.
var logger *zap.Logger = zap.NewNop()

func SetLogger(l *zap.Logger) {
	logger = l
}

// WriteError is the single place that turns a Go error into an HTTP response.
// Handlers never set status codes for domain errors themselves — they just
// call this. Anything that isn't an *apperr.AppError is treated as an
// unexpected internal error (client gets a generic 500, no leaked detail).
func WriteError(w http.ResponseWriter, err error) {
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		appErr = apperr.NewInternalError(err)
	}

	if appErr.Status >= 500 {
		logger.Error("request failed", zap.Error(appErr.Err), zap.String("message", appErr.Message))
	}

	WriteJSON(w, appErr.Status, errorResponse{
		Code:    appErr.Code,
		Message: appErr.Message,
		Fields:  appErr.Fields,
	})
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
