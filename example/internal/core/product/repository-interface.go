package product

import (
	"context"

	"github.com/behzade/httprpc/example/internal/domain"
)

// Repository defines storage dependencies for the product module.
type Repository interface {
	List(ctx context.Context, in domain.ListProductsInput) ([]domain.Product, int, error)
	Get(ctx context.Context, id string) (*domain.Product, error)
	Create(ctx context.Context, in domain.CreateProductInput) (*domain.Product, error)
	Update(ctx context.Context, in domain.UpdateProductInput) (*domain.Product, error)
	Delete(ctx context.Context, id string) error
}
