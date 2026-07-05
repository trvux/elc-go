package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/settings/application"
	"github.com/trvux/elc-go/internal/settings/domain"
)

type SettingsHandler struct {
	repo domain.SettingsRepository
}

func NewSettingsHandler(repo domain.SettingsRepository) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

func (h *SettingsHandler) List(w http.ResponseWriter, r *http.Request) {
	settings, err := application.GetSettings(r.Context(), h.repo)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toSiteSettingDTOList(settings))
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dtos []SiteSettingDTO
	if err := json.NewDecoder(r.Body).Decode(&dtos); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	settings, err := toSiteSettingDomainList(dtos)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	if err := application.UpdateSettings(r.Context(), h.repo, settings); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
