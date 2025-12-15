package domain

// Product represents a product entity.
type Product struct {
	ID          string
	Name        string
	Description string
	Price       float64
	Stock       int
	UpdatedAt   int64
}

// ListProductsInput defines filtering, sorting, and pagination parameters.
type ListProductsInput struct {
	Page          int
	PerPage       int
	SortField     string
	SortDirection string
	Query         string
}

// ListProductsResult wraps the list output.
type ListProductsResult struct {
	Items []Product
	Total int
}

// CreateProductInput is the payload for creating a product.
type CreateProductInput struct {
	Name        string
	Description string
	Price       float64
	Stock       int
}

// UpdateProductInput is the payload for updating a product.
type UpdateProductInput struct {
	ID          string
	Name        string
	Description string
	Price       float64
	Stock       int
}
