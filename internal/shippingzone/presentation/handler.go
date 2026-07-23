package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/shippingzone/application"
	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

type ShippingZoneHandler struct {
	zoneRepo     domain.ShippingZoneRepository
	provinceRepo domain.ProvinceRepository
	wardRepo     domain.WardRepository
}

func NewShippingZoneHandler(zoneRepo domain.ShippingZoneRepository, provinceRepo domain.ProvinceRepository, wardRepo domain.WardRepository) *ShippingZoneHandler {
	return &ShippingZoneHandler{zoneRepo: zoneRepo, provinceRepo: provinceRepo, wardRepo: wardRepo}
}

func (h *ShippingZoneHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.ZoneFilter{IncludeDeleted: r.URL.Query().Get("include_deleted") == "true"}

	zones, err := application.GetZones(r.Context(), h.zoneRepo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toZoneResponseList(zones))
}

func (h *ShippingZoneHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	z, err := application.GetZoneByID(r.Context(), h.zoneRepo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if z == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("shipping zone"))
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toZoneResponse(z))
}

func (h *ShippingZoneHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	z, err := application.CreateZone(r.Context(), h.zoneRepo, domain.CreateZoneInput{
		Name:          req.Name,
		FeeVND:        req.FeeVND,
		MinDays:       req.MinDays,
		MaxDays:       req.MaxDays,
		IsDefault:     req.IsDefault,
		ProvinceCodes: req.ProvinceCodes,
		WardCodes:     req.WardCodes,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, toZoneResponse(z))
}

func (h *ShippingZoneHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	z, err := application.UpdateZone(r.Context(), h.zoneRepo, domain.UpdateZoneInput{
		ID:            id,
		Name:          req.Name,
		FeeVND:        req.FeeVND,
		MinDays:       req.MinDays,
		MaxDays:       req.MaxDays,
		IsDefault:     req.IsDefault,
		ProvinceCodes: req.ProvinceCodes,
		WardCodes:     req.WardCodes,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toZoneResponse(z))
}

func (h *ShippingZoneHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteZone(r.Context(), h.zoneRepo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ShippingZoneHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	province := r.URL.Query().Get("province")
	if province == "" {
		httpserver.WriteError(w, apperr.NewValidationError("province query param is required", nil))
		return
	}
	ward := r.URL.Query().Get("ward")

	z, err := application.LookupZone(r.Context(), h.zoneRepo, province, ward)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if z == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("shipping zone"))
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toLookupResponse(z))
}

func (h *ShippingZoneHandler) Default(w http.ResponseWriter, r *http.Request) {
	z, err := application.GetDefaultZone(r.Context(), h.zoneRepo)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if z == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("shipping zone"))
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toLookupResponse(z))
}

func (h *ShippingZoneHandler) ListProvinces(w http.ResponseWriter, r *http.Request) {
	provinces, err := application.GetProvinces(r.Context(), h.provinceRepo)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toProvinceResponseList(provinces))
}

func (h *ShippingZoneHandler) CreateProvince(w http.ResponseWriter, r *http.Request) {
	var req createProvinceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	p, err := application.CreateProvince(r.Context(), h.provinceRepo, req.Code, req.Name)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, toProvinceResponse(p))
}

func (h *ShippingZoneHandler) UpdateProvince(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	var req updateProvinceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	p, err := application.UpdateProvince(r.Context(), h.provinceRepo, code, req.Name)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toProvinceResponse(p))
}

func (h *ShippingZoneHandler) ListWards(w http.ResponseWriter, r *http.Request) {
	province := r.URL.Query().Get("province")
	if province == "" {
		httpserver.WriteError(w, apperr.NewValidationError("province query param is required", nil))
		return
	}

	wards, err := application.GetWardsByProvince(r.Context(), h.wardRepo, province)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toWardResponseList(wards))
}

func (h *ShippingZoneHandler) DeleteProvince(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	if err := application.DeleteProvince(r.Context(), h.provinceRepo, code); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
