package domain

import (
	"context"
	"time"
)

// Product es la información genérica de un artículo (escaparate). El stock y
// los precios reales viven en ProductVariant.
type Product struct {
	ID          string
	CategoryID  *string
	Name        string
	Description string
	Images      []string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProductVariant resuelve el stock atómico y los precios específicos por
// combinación de atributos (por ejemplo "Rojo / Talle S"). El precio B2B es
// opcional (mayorista).
type ProductVariant struct {
	ID            string
	ProductID     string
	SKU           string
	Name          string
	Price         float64
	B2BPrice      *float64
	StockQuantity int
	Attributes    map[string]any
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ProductRepository es el puerto de persistencia para productos y variantes.
// Maneja ambas entidades de forma atómica (alta de variantes con su stock y
// ajustes de stock concurrentes vía operaciones atómicas en SQL).
type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
	List(ctx context.Context) ([]Product, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id string) error

	CreateVariant(ctx context.Context, variant *ProductVariant) error
	GetVariantByID(ctx context.Context, id string) (*ProductVariant, error)
	ListVariantsByProduct(ctx context.Context, productID string) ([]ProductVariant, error)
	UpdateVariant(ctx context.Context, variant *ProductVariant) error
	DeleteVariant(ctx context.Context, id string) error
	// AdjustStockQuantity suma delta al stock de forma atómica. Retorna un
	// error de validación si el resultado fuese negativo.
	AdjustStockQuantity(ctx context.Context, id string, delta int) (*ProductVariant, error)
}
