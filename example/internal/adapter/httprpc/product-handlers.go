package httprpcadapter

import (
	"context"
	"errors"

	"github.com/behzade/httprpc"
	"github.com/behzade/httprpc/example/internal/core/product"
	"github.com/behzade/httprpc/example/internal/domain"
)

// ProductHandlers wires httprpc endpoints to the product module.
type ProductHandlers struct {
	module *product.Module
}

func NewProductHandlers(module *product.Module) *ProductHandlers {
	return &ProductHandlers{module: module}
}

// HTTP DTOs (only used at the transport layer).
type (
	ListProductsRequest struct {
		Page          int    `json:"page" query:"page"`
		PerPage       int    `json:"per_page" query:"per_page"`
		SortField     string `json:"sort_field" query:"sort_field"`
		SortDirection string `json:"sort_direction" query:"sort_direction"`
		Query         string `json:"query" query:"query"`
	}

	ListProductsResponse struct {
		Items []Product `json:"items"`
		Total int       `json:"total"`
	}

	GetProductRequest struct {
		ID string `json:"id" query:"id"`
	}

	CreateProductRequest struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int     `json:"stock"`
	}

	UpdateProductRequest struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int     `json:"stock"`
	}

	DeleteProductRequest struct {
		ID string `json:"id"`
	}

	DeleteProductResponse struct {
		ID string `json:"id"`
	}

	Product struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int     `json:"stock"`
		UpdatedAt   int64   `json:"updated_at"`
	}
)

// Register mounts product endpoints under the provided group.
func (h *ProductHandlers) Register(api *httprpc.EndpointGroup) {
	httprpc.RegisterHandler(
		api,
		httprpc.GET(
			func(ctx context.Context, req ListProductsRequest) (ListProductsResponse, error) {
				result, err := h.module.List(ctx, domain.ListProductsInput(req))
				if err != nil {
					return ListProductsResponse{}, mapError(err)
				}
				return ListProductsResponse{
					Items: toProductDTOs(result.Items),
					Total: result.Total,
				}, nil
			},
			"/products/list",
		),
	)

	httprpc.RegisterHandler(
		api,
		httprpc.GET(
			func(ctx context.Context, req GetProductRequest) (Product, error) {
				p, err := h.module.Get(ctx, req.ID)
				if err != nil {
					return Product{}, mapError(err)
				}
				return toProductDTO(*p), nil
			},
			"/products/get",
		),
	)

	httprpc.RegisterHandler(
		api,
		httprpc.POST(
			func(ctx context.Context, req CreateProductRequest) (Product, error) {
				p, err := h.module.Create(ctx, domain.CreateProductInput(req))
				if err != nil {
					return Product{}, mapError(err)
				}
				return toProductDTO(*p), nil
			},
			"/products",
		),
	)

	httprpc.RegisterHandler(
		api,
		httprpc.PUT(
			func(ctx context.Context, req UpdateProductRequest) (Product, error) {
				p, err := h.module.Update(ctx, domain.UpdateProductInput(req))
				if err != nil {
					return Product{}, mapError(err)
				}
				return toProductDTO(*p), nil
			},
			"/products",
		),
	)

	httprpc.RegisterHandler(
		api,
		httprpc.DELETE(
			func(ctx context.Context, req DeleteProductRequest) (DeleteProductResponse, error) {
				if err := h.module.Delete(ctx, req.ID); err != nil {
					return DeleteProductResponse{}, mapError(err)
				}
				return DeleteProductResponse{ID: req.ID}, nil
			},
			"/products",
		),
	)
}

func toProductDTO(p domain.Product) Product {
	return Product{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toProductDTOs(items []domain.Product) []Product {
	out := make([]Product, 0, len(items))
	for _, p := range items {
		out = append(out, toProductDTO(p))
	}
	return out
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		return httprpc.StatusError{Status: 400, Err: err}
	case errors.Is(err, domain.ErrNotFound):
		return httprpc.StatusError{Status: 404, Err: err}
	default:
		return err
	}
}
