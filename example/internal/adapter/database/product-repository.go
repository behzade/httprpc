package database

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/behzade/httprpc/example/internal/domain"
)

// InMemoryProductRepository is a simple in-memory store implementing the domain.ProductRepository.
type InMemoryProductRepository struct {
	mu       sync.RWMutex
	nextID   int
	products map[string]domain.Product
}

func NewInMemoryProductRepository() *InMemoryProductRepository {
	now := time.Now().Unix()
	products := map[string]domain.Product{
		"p-1": {
			ID:          "p-1",
			Name:        "Vintage Desk Lamp",
			Description: "A sturdy brass lamp with adjustable arm and warm Edison bulb.",
			Price:       89.50,
			Stock:       12,
			UpdatedAt:   now,
		},
		"p-2": {
			ID:          "p-2",
			Name:        "Noise-Canceling Headphones",
			Description: "Over-ear headphones with adaptive ANC and 30-hour battery life.",
			Price:       249.00,
			Stock:       34,
			UpdatedAt:   now,
		},
		"p-3": {
			ID:          "p-3",
			Name:        "Ergonomic Office Chair",
			Description: "Mesh back, lumbar support, and adjustable height for long work sessions.",
			Price:       319.00,
			Stock:       8,
			UpdatedAt:   now,
		},
	}

	return &InMemoryProductRepository{
		nextID:   len(products) + 1,
		products: products,
	}
}

func (r *InMemoryProductRepository) List(ctx context.Context, in domain.ListProductsInput) ([]domain.Product, int, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := strings.ToLower(strings.TrimSpace(in.Query))
	items := make([]domain.Product, 0, len(r.products))
	for _, p := range r.products {
		if query == "" || strings.Contains(strings.ToLower(p.Name), query) || strings.Contains(strings.ToLower(p.Description), query) {
			items = append(items, p)
		}
	}

	sortField := in.SortField
	sortDirection := in.SortDirection
	less := func(i, j int) bool {
		switch sortField {
		case "price":
			if sortDirection == "ASC" {
				return items[i].Price < items[j].Price
			}
			return items[i].Price > items[j].Price
		case "stock":
			if sortDirection == "ASC" {
				return items[i].Stock < items[j].Stock
			}
			return items[i].Stock > items[j].Stock
		case "updated_at", "updatedat":
			if sortDirection == "ASC" {
				return items[i].UpdatedAt < items[j].UpdatedAt
			}
			return items[i].UpdatedAt > items[j].UpdatedAt
		default:
			if sortDirection == "ASC" {
				return items[i].Name < items[j].Name
			}
			return items[i].Name > items[j].Name
		}
	}
	sort.SliceStable(items, less)

	total := len(items)
	start := (in.Page - 1) * in.PerPage
	if start > total {
		start = total
	}
	end := start + in.PerPage
	if end > total {
		end = total
	}

	return append([]domain.Product(nil), items[start:end]...), total, nil
}

func (r *InMemoryProductRepository) Get(ctx context.Context, id string) (*domain.Product, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		return nil, fmt.Errorf("%w: product %q not found", domain.ErrNotFound, id)
	}
	return &product, nil
}

func (r *InMemoryProductRepository) Create(ctx context.Context, in domain.CreateProductInput) (*domain.Product, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	id := fmt.Sprintf("p-%d", r.nextID)
	r.nextID++

	p := domain.Product{
		ID:          id,
		Name:        in.Name,
		Description: in.Description,
		Price:       in.Price,
		Stock:       in.Stock,
		UpdatedAt:   time.Now().Unix(),
	}
	r.products[id] = p
	return &p, nil
}

func (r *InMemoryProductRepository) Update(ctx context.Context, in domain.UpdateProductInput) (*domain.Product, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.products[in.ID]
	if !ok {
		return nil, fmt.Errorf("%w: product %q not found", domain.ErrNotFound, in.ID)
	}

	existing.Name = in.Name
	existing.Description = in.Description
	existing.Price = in.Price
	existing.Stock = in.Stock
	existing.UpdatedAt = time.Now().Unix()

	r.products[in.ID] = existing
	return &existing, nil
}

func (r *InMemoryProductRepository) Delete(ctx context.Context, id string) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.products[id]; !ok {
		return fmt.Errorf("%w: product %q not found", domain.ErrNotFound, id)
	}
	delete(r.products, id)
	return nil
}
