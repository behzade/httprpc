package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/behzade/httprpc"
)

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	UpdatedAt   int64   `json:"updated_at"`
}

type ListProductsRequest struct {
	Page          int    `json:"page"`
	PerPage       int    `json:"per_page"`
	SortField     string `json:"sort_field"`
	SortDirection string `json:"sort_direction"`
	Query         string `json:"query"`
}

type ListProductsResponse struct {
	Items []Product `json:"items"`
	Total int       `json:"total"`
}

type GetProductRequest struct {
	ID string `json:"id"`
}

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

type UpdateProductRequest struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

type DeleteProductRequest struct {
	ID string `json:"id"`
}

type DeleteProductResponse struct {
	ID string `json:"id"`
}

type productRepo struct {
	mu       sync.RWMutex
	nextID   int
	products map[string]Product
}

func newProductRepo() *productRepo {
	now := time.Now().Unix()
	products := map[string]Product{
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

	return &productRepo{
		nextID:   len(products) + 1,
		products: products,
	}
}

func (r *productRepo) List(req ListProductsRequest) (ListProductsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	page := req.Page
	if page < 1 {
		page = 1
	}
	perPage := req.PerPage
	if perPage < 1 {
		perPage = 25
	}

	query := strings.TrimSpace(strings.ToLower(req.Query))
	items := make([]Product, 0, len(r.products))
	for _, p := range r.products {
		if query == "" || strings.Contains(strings.ToLower(p.Name), query) || strings.Contains(strings.ToLower(p.Description), query) {
			items = append(items, p)
		}
	}

	sortField := strings.ToLower(req.SortField)
	if sortField == "" {
		sortField = "name"
	}
	sortDirection := strings.ToUpper(req.SortDirection)
	if sortDirection != "DESC" {
		sortDirection = "ASC"
	}
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
	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}

	return ListProductsResponse{
		Items: append([]Product(nil), items[start:end]...),
		Total: total,
	}, nil
}

func (r *productRepo) Get(id string) (Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		return Product{}, httprpc.StatusError{Status: 404, Err: fmt.Errorf("product %q not found", id)}
	}
	return product, nil
}

func (r *productRepo) Create(req CreateProductRequest) (Product, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" {
		return Product{}, httprpc.StatusError{Status: 400, Err: fmt.Errorf("name is required")}
	}
	if req.Price < 0 {
		return Product{}, httprpc.StatusError{Status: 400, Err: fmt.Errorf("price must be non-negative")}
	}
	if req.Stock < 0 {
		return Product{}, httprpc.StatusError{Status: 400, Err: fmt.Errorf("stock must be non-negative")}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := fmt.Sprintf("p-%d", r.nextID)
	r.nextID++

	product := Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		UpdatedAt:   time.Now().Unix(),
	}
	r.products[id] = product
	return product, nil
}

func (r *productRepo) Update(req UpdateProductRequest) (Product, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.ID == "" {
		return Product{}, httprpc.StatusError{Status: 400, Err: fmt.Errorf("id is required")}
	}
	if req.Name == "" {
		return Product{}, httprpc.StatusError{Status: 400, Err: fmt.Errorf("name is required")}
	}
	if req.Price < 0 {
		return Product{}, httprpc.StatusError{Status: 400, Err: fmt.Errorf("price must be non-negative")}
	}
	if req.Stock < 0 {
		return Product{}, httprpc.StatusError{Status: 400, Err: fmt.Errorf("stock must be non-negative")}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.products[req.ID]
	if !ok {
		return Product{}, httprpc.StatusError{Status: 404, Err: fmt.Errorf("product %q not found", req.ID)}
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Price = req.Price
	existing.Stock = req.Stock
	existing.UpdatedAt = time.Now().Unix()

	r.products[req.ID] = existing
	return existing, nil
}

func (r *productRepo) Delete(id string) (DeleteProductResponse, error) {
	if id == "" {
		return DeleteProductResponse{}, httprpc.StatusError{Status: 400, Err: fmt.Errorf("id is required")}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.products[id]; !ok {
		return DeleteProductResponse{}, httprpc.StatusError{Status: 404, Err: fmt.Errorf("product %q not found", id)}
	}

	delete(r.products, id)
	return DeleteProductResponse{ID: id}, nil
}
