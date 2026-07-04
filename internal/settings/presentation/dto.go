package presentation

import "github.com/trvux/elc-go/internal/settings/domain"

type SiteSettingDTO struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func toSiteSettingDTOList(settings []*domain.SiteSetting) []SiteSettingDTO {
	result := make([]SiteSettingDTO, len(settings))
	for i, s := range settings {
		result[i] = SiteSettingDTO{
			Key:   s.Key(),
			Value: s.Value(),
		}
	}
	return result
}

func toSiteSettingDomainList(dtos []SiteSettingDTO) ([]*domain.SiteSetting, error) {
	result := make([]*domain.SiteSetting, len(dtos))
	for i, dto := range dtos {
		s, err := domain.NewSiteSetting(dto.Key, dto.Value)
		if err != nil {
			return nil, err
		}
		result[i] = s
	}
	return result, nil
}
