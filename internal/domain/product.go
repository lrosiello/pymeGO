package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PriceMode indica qué lista de precios usa el comercio. Al ser una aplicación
// single-tenant, se configura una sola vez por instancia y se aplica tanto a la
// vitrina pública como al cálculo del total de los pedidos.
type PriceMode string

const (
	// PriceModeB2C usa el precio minorista (ProductVariant.Price).
	PriceModeB2C PriceMode = "b2c"
	// PriceModeB2B usa el precio mayorista (ProductVariant.B2BPrice), con
	// fallback al minorista cuando no está definido.
	PriceModeB2B PriceMode = "b2b"
)

// ParsePriceMode interpreta el modo de precios. Un valor vacío se resuelve como
// b2c para conservar el comportamiento histórico.
func ParsePriceMode(raw string) (PriceMode, error) {
	switch PriceMode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", PriceModeB2C:
		return PriceModeB2C, nil
	case PriceModeB2B:
		return PriceModeB2B, nil
	default:
		return "", fmt.Errorf("modo de precios inválido %q: debe ser b2c o b2b", raw)
	}
}

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

// PriceForMode devuelve el precio efectivo según la lista configurada. En modo
// B2B, si la variante no tiene precio mayorista, cae al precio minorista.
func (v *ProductVariant) PriceForMode(mode PriceMode) float64 {
	if mode == PriceModeB2B && v.B2BPrice != nil {
		return *v.B2BPrice
	}
	return v.Price
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

	// ListSellable devuelve productos activos con al menos una variante activa
	// con stock disponible, opcionalmente filtrados por categoría.
	ListSellable(ctx context.Context, categoryID *string) ([]Product, error)
	// GetSellableByID devuelve un producto vendible (activo y con stock) o
	// ErrNotFound si no cumple las condiciones de la vitrina pública.
	GetSellableByID(ctx context.Context, id string) (*Product, error)
	// ListActiveVariantsByProduct devuelve las variantes activas de un producto.
	ListActiveVariantsByProduct(ctx context.Context, productID string) ([]ProductVariant, error)
}
