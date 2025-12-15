// Package product implements the product use-cases.
package product

import (
	"context"
	"fmt"
	"strings"

	"github.com/behzade/httprpc/example/internal/domain"
)

// Module exposes product operations.
type Module struct {
	repo Repository
}

func New(repo Repository) *Module {
	return &Module{repo: repo}
}

func (m *Module) List(ctx context.Context, in domain.ListProductsInput) (domain.ListProductsResult, error) {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PerPage < 1 {
		in.PerPage = 25
	}
	if in.SortField == "" {
		in.SortField = "name"
	}
	in.SortField = strings.ToLower(in.SortField)
	in.SortDirection = strings.ToUpper(in.SortDirection)
	if in.SortDirection != "DESC" {
		in.SortDirection = "ASC"
	}
	in.Query = strings.TrimSpace(in.Query)

	items, total, err := m.repo.List(ctx, in)
	if err != nil {
		return domain.ListProductsResult{}, err
	}
	return domain.ListProductsResult{Items: items, Total: total}, nil
}

func (m *Module) Get(ctx context.Context, id string) (*domain.Product, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", domain.ErrInvalidArgument)
	}
	return m.repo.Get(ctx, id)
}

func (m *Module) Create(ctx context.Context, in domain.CreateProductInput) (*domain.Product, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidArgument)
	}
	if in.Price < 0 {
		return nil, fmt.Errorf("%w: price must be non-negative", domain.ErrInvalidArgument)
	}
	if in.Stock < 0 {
		return nil, fmt.Errorf("%w: stock must be non-negative", domain.ErrInvalidArgument)
	}
	return m.repo.Create(ctx, in)
}

func (m *Module) Update(ctx context.Context, in domain.UpdateProductInput) (*domain.Product, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.ID == "" {
		return nil, fmt.Errorf("%w: id is required", domain.ErrInvalidArgument)
	}
	if in.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidArgument)
	}
	if in.Price < 0 {
		return nil, fmt.Errorf("%w: price must be non-negative", domain.ErrInvalidArgument)
	}
	if in.Stock < 0 {
		return nil, fmt.Errorf("%w: stock must be non-negative", domain.ErrInvalidArgument)
	}
	return m.repo.Update(ctx, in)
}

func (m *Module) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: id is required", domain.ErrInvalidArgument)
	}
	return m.repo.Delete(ctx, id)
}
