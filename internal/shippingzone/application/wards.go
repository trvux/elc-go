package application

import (
	"context"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

func GetWardsByProvince(ctx context.Context, repo domain.WardRepository, provinceCode string) ([]*domain.Ward, error) {
	return repo.GetByProvince(ctx, provinceCode)
}
