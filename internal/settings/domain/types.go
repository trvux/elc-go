package domain

import "github.com/trvux/elc-go/internal/platform/apperr"

type SiteSetting struct {
	key   string
	value string
}

func NewSiteSetting(key, value string) (*SiteSetting, error) {
	if key == "" {
		return nil, apperr.NewValidationError("key cannot be empty", map[string][]string{
			"key": {"key cannot be empty"},
		})
	}
	return &SiteSetting{
		key:   key,
		value: value,
	}, nil
}

func RehydrateSiteSetting(key, value string) *SiteSetting {
	return &SiteSetting{
		key:   key,
		value: value,
	}
}

func (s *SiteSetting) Key() string   { return s.key }
func (s *SiteSetting) Value() string { return s.value }

func (s *SiteSetting) UpdateValue(val string) {
	s.value = val
}
