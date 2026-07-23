package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

func GetProvinces(ctx context.Context, repo domain.ProvinceRepository) ([]*domain.Province, error) {
	return repo.GetAll(ctx)
}

func CreateProvince(ctx context.Context, repo domain.ProvinceRepository, code, name string) (*domain.Province, error) {
	if code == "" || name == "" {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{
			"code": {"code is required"},
			"name": {"name is required"},
		})
	}
	return repo.Create(ctx, &domain.Province{Code: code, Name: name})
}

func UpdateProvince(ctx context.Context, repo domain.ProvinceRepository, code, name string) (*domain.Province, error) {
	if name == "" {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"name": {"name is required"}})
	}
	return repo.Update(ctx, code, name)
}

func DeleteProvince(ctx context.Context, repo domain.ProvinceRepository, code string) error {
	return repo.Delete(ctx, code)
}
