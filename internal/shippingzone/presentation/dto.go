package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

type zoneResponse struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	FeeVND        int64      `json:"fee_vnd"`
	MinDays       int        `json:"min_days"`
	MaxDays       int        `json:"max_days"`
	IsDefault     bool       `json:"is_default"`
	ProvinceCodes []string   `json:"province_codes"`
	WardCodes     []string   `json:"ward_codes"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
}

func toZoneResponse(z *domain.ShippingZone) zoneResponse {
	provinceCodes := z.ProvinceCodes()
	if provinceCodes == nil {
		provinceCodes = []string{}
	}
	wardCodes := z.WardCodes()
	if wardCodes == nil {
		wardCodes = []string{}
	}
	return zoneResponse{
		ID:            z.ID(),
		Name:          z.Name(),
		FeeVND:        z.FeeVND(),
		MinDays:       z.MinDays(),
		MaxDays:       z.MaxDays(),
		IsDefault:     z.IsDefault(),
		ProvinceCodes: provinceCodes,
		WardCodes:     wardCodes,
		CreatedAt:     z.CreatedAt(),
		UpdatedAt:     z.UpdatedAt(),
		DeletedAt:     z.DeletedAt(),
	}
}

func toZoneResponseList(zones []*domain.ShippingZone) []zoneResponse {
	result := make([]zoneResponse, len(zones))
	for i, z := range zones {
		result[i] = toZoneResponse(z)
	}
	return result
}

type createZoneRequest struct {
	Name          string   `json:"name"`
	FeeVND        int64    `json:"fee_vnd"`
	MinDays       int      `json:"min_days"`
	MaxDays       int      `json:"max_days"`
	IsDefault     bool     `json:"is_default"`
	ProvinceCodes []string `json:"province_codes"`
	WardCodes     []string `json:"ward_codes"`
}

type updateZoneRequest struct {
	Name          *string  `json:"name"`
	FeeVND        *int64   `json:"fee_vnd"`
	MinDays       *int     `json:"min_days"`
	MaxDays       *int     `json:"max_days"`
	IsDefault     *bool    `json:"is_default"`
	ProvinceCodes []string `json:"province_codes"`
	WardCodes     []string `json:"ward_codes"`
}

type provinceResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func toProvinceResponse(p *domain.Province) provinceResponse {
	return provinceResponse{Code: p.Code, Name: p.Name}
}

func toProvinceResponseList(provinces []*domain.Province) []provinceResponse {
	result := make([]provinceResponse, len(provinces))
	for i, p := range provinces {
		result[i] = toProvinceResponse(p)
	}
	return result
}

type createProvinceRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type updateProvinceRequest struct {
	Name string `json:"name"`
}

type wardResponse struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	ProvinceCode string `json:"province_code"`
}

func toWardResponse(w *domain.Ward) wardResponse {
	return wardResponse{Code: w.Code, Name: w.Name, ProvinceCode: w.ProvinceCode}
}

func toWardResponseList(wards []*domain.Ward) []wardResponse {
	result := make([]wardResponse, len(wards))
	for i, w := range wards {
		result[i] = toWardResponse(w)
	}
	return result
}

type lookupResponse struct {
	ZoneName string `json:"zone_name"`
	FeeVND   int64  `json:"fee_vnd"`
	MinDays  int    `json:"min_days"`
	MaxDays  int    `json:"max_days"`
}

func toLookupResponse(z *domain.ShippingZone) lookupResponse {
	return lookupResponse{
		ZoneName: z.Name(),
		FeeVND:   z.FeeVND(),
		MinDays:  z.MinDays(),
		MaxDays:  z.MaxDays(),
	}
}
