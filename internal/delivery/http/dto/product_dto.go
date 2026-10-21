package dto

import (
	"time"

	"pymego/internal/domain"
	"pymego/internal/usecase"
)

// ---------------------------------------------------------------------------
// Categorías
// ---------------------------------------------------------------------------

// CreateCategoryRequest es el cuerpo de POST /api/v1/categories.
type CreateCategoryRequest struct {
	ParentID    *string `json:"parent_id"`
	Name        string  `json:"name" binding:"required,min=2,max=100"`
	Description string  `json:"description" binding:"max=2000"`
}

// ToInput traduce el DTO al input del caso de uso.
func (r CreateCategoryRequest) ToInput() usecase.CreateCategoryInput {
	return usecase.CreateCategoryInput{
		ParentID:    r.ParentID,
		Name:        r.Name,
		Description: r.Description,
	}
}

// UpdateCategoryRequest es el cuerpo de PUT /api/v1/categories/:id.
type UpdateCategoryRequest struct {
	ParentID    *string `json:"parent_id"`
	Name        string  `json:"name" binding:"required,min=2,max=100"`
	Description string  `json:"description" binding:"max=2000"`
	IsActive    *bool   `json:"is_active"`
}

// ToInput traduce el DTO al input del caso de uso.
func (r UpdateCategoryRequest) ToInput() usecase.UpdateCategoryInput {
	return usecase.UpdateCategoryInput{
		ParentID:    r.ParentID,
		Name:        r.Name,
		Description: r.Description,
		IsActive:    boolOrDefault(r.IsActive, true),
	}
}

// CategoryResponse es la representación pública de una categoría.
type CategoryResponse struct {
	ID          string    `json:"id"`
	ParentID    *string   `json:"parent_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewCategoryResponse construye la respuesta desde la entidad de dominio.
func NewCategoryResponse(category *domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:          category.ID,
		ParentID:    category.ParentID,
		Name:        category.Name,
		Description: category.Description,
		IsActive:    category.IsActive,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Productos
// ---------------------------------------------------------------------------

// CreateProductRequest es el cuerpo de POST /api/v1/products. Admite el alta
// atómica de variantes iniciales.
type CreateProductRequest struct {
	CategoryID  *string                `json:"category_id"`
	Name        string                 `json:"name" binding:"required,min=2,max=200"`
	Description string                 `json:"description" binding:"max=5000"`
	Images      []string               `json:"images"`
	IsActive    *bool                  `json:"is_active"`
	Variants    []CreateVariantRequest `json:"variants" binding:"dive"`
}

// ToInput traduce el DTO al input del caso de uso.
func (r CreateProductRequest) ToInput() usecase.CreateProductInput {
	variants := make([]usecase.CreateVariantInput, 0, len(r.Variants))
	for _, v := range r.Variants {
		variants = append(variants, v.ToInput())
	}
	return usecase.CreateProductInput{
		CategoryID:  r.CategoryID,
		Name:        r.Name,
		Description: r.Description,
		Images:      r.Images,
		IsActive:    boolOrDefault(r.IsActive, true),
		Variants:    variants,
	}
}

// UpdateProductRequest es el cuerpo de PUT /api/v1/products/:id.
type UpdateProductRequest struct {
	CategoryID  *string  `json:"category_id"`
	Name        string   `json:"name" binding:"required,min=2,max=200"`
	Description string   `json:"description" binding:"max=5000"`
	Images      []string `json:"images"`
	IsActive    *bool    `json:"is_active"`
}

// ToInput traduce el DTO al input del caso de uso.
func (r UpdateProductRequest) ToInput() usecase.UpdateProductInput {
	return usecase.UpdateProductInput{
		CategoryID:  r.CategoryID,
		Name:        r.Name,
		Description: r.Description,
		Images:      r.Images,
		IsActive:    boolOrDefault(r.IsActive, true),
	}
}

// ProductResponse es la representación pública de un producto con sus variantes.
type ProductResponse struct {
	ID          string            `json:"id"`
	CategoryID  *string           `json:"category_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Images      []string          `json:"images"`
	IsActive    bool              `json:"is_active"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Variants    []VariantResponse `json:"variants"`
}

// NewProductResponse construye la respuesta (producto + variantes) desde el
// detalle retornado por el caso de uso.
func NewProductResponse(detail usecase.ProductDetail) ProductResponse {
	variants := make([]VariantResponse, 0, len(detail.Variants))
	for i := range detail.Variants {
		variants = append(variants, NewVariantResponse(&detail.Variants[i]))
	}

	return ProductResponse{
		ID:          detail.Product.ID,
		CategoryID:  detail.Product.CategoryID,
		Name:        detail.Product.Name,
		Description: detail.Product.Description,
		Images:      detail.Product.Images,
		IsActive:    detail.Product.IsActive,
		CreatedAt:   detail.Product.CreatedAt,
		UpdatedAt:   detail.Product.UpdatedAt,
		Variants:    variants,
	}
}

// ---------------------------------------------------------------------------
// Variantes
// ---------------------------------------------------------------------------

// CreateVariantRequest es el cuerpo de POST /api/v1/products/:id/variants.
type CreateVariantRequest struct {
	SKU           string         `json:"sku" binding:"required,min=1,max=100"`
	Name          string         `json:"name" binding:"required,min=1,max=150"`
	Price         float64        `json:"price" binding:"required,gte=0"`
	B2BPrice      *float64       `json:"b2b_price" binding:"omitempty,gte=0"`
	StockQuantity int            `json:"stock_quantity" binding:"gte=0"`
	Attributes    map[string]any `json:"attributes"`
}

// ToInput traduce el DTO al input del caso de uso.
func (r CreateVariantRequest) ToInput() usecase.CreateVariantInput {
	return usecase.CreateVariantInput{
		SKU:           r.SKU,
		Name:          r.Name,
		Price:         r.Price,
		B2BPrice:      r.B2BPrice,
		StockQuantity: r.StockQuantity,
		Attributes:    r.Attributes,
	}
}

// UpdateVariantRequest es el cuerpo de PUT /api/v1/variants/:id.
type UpdateVariantRequest struct {
	SKU        string         `json:"sku" binding:"required,min=1,max=100"`
	Name       string         `json:"name" binding:"required,min=1,max=150"`
	Price      float64        `json:"price" binding:"required,gte=0"`
	B2BPrice   *float64       `json:"b2b_price" binding:"omitempty,gte=0"`
	Attributes map[string]any `json:"attributes"`
	IsActive   *bool          `json:"is_active"`
}

// ToInput traduce el DTO al input del caso de uso.
func (r UpdateVariantRequest) ToInput() usecase.UpdateVariantInput {
	return usecase.UpdateVariantInput{
		SKU:        r.SKU,
		Name:       r.Name,
		Price:      r.Price,
		B2BPrice:   r.B2BPrice,
		Attributes: r.Attributes,
		IsActive:   boolOrDefault(r.IsActive, true),
	}
}

// AdjustStockRequest es el cuerpo de PATCH /api/v1/variants/:id/stock.
type AdjustStockRequest struct {
	Delta int `json:"delta" binding:"required,ne=0"`
}

// VariantResponse es la representación pública de una variante.
type VariantResponse struct {
	ID            string         `json:"id"`
	ProductID     string         `json:"product_id"`
	SKU           string         `json:"sku"`
	Name          string         `json:"name"`
	Price         float64        `json:"price"`
	B2BPrice      *float64       `json:"b2b_price"`
	StockQuantity int            `json:"stock_quantity"`
	Attributes    map[string]any `json:"attributes"`
	IsActive      bool           `json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// NewVariantResponse construye la respuesta desde la entidad de dominio.
func NewVariantResponse(v *domain.ProductVariant) VariantResponse {
	attributes := v.Attributes
	if attributes == nil {
		attributes = map[string]any{}
	}
	return VariantResponse{
		ID:            v.ID,
		ProductID:     v.ProductID,
		SKU:           v.SKU,
		Name:          v.Name,
		Price:         v.Price,
		B2BPrice:      v.B2BPrice,
		StockQuantity: v.StockQuantity,
		Attributes:    attributes,
		IsActive:      v.IsActive,
		CreatedAt:     v.CreatedAt,
		UpdatedAt:     v.UpdatedAt,
	}
}

// boolOrDefault devuelve el valor del puntero o un valor por defecto si es nil.
func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
