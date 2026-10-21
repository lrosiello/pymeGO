package usecase

import (
	"context"

	"pymego/internal/domain"
)

// CatalogUseCase expone la lectura pública de la vitrina: solo categorías y
// productos activos, con los precios resueltos según el modo configurado.
type CatalogUseCase interface {
	ListCategories(ctx context.Context) ([]domain.Category, error)
	ListProducts(ctx context.Context, categoryID *string) ([]CatalogProduct, error)
	GetProduct(ctx context.Context, id string) (*CatalogProduct, error)
}

// CatalogProduct agrupa un producto con las variantes visibles en la vitrina.
type CatalogProduct struct {
	Product  *domain.Product
	Variants []CatalogVariant
}

// CatalogVariant es la proyección pública de una variante: expone únicamente el
// precio efectivo (B2C o B2B según configuración), evitando filtrar la otra
// lista de precios.
type CatalogVariant struct {
	ID            string
	Name          string
	Price         float64
	StockQuantity int
	Available     bool
	Attributes    map[string]any
}

// catalogUseCase implementa CatalogUseCase.
type catalogUseCase struct {
	categories domain.CategoryRepository
	products   domain.ProductRepository
	priceMode  domain.PriceMode
}

// NewCatalogUseCase construye el caso de uso de la vitrina pública.
func NewCatalogUseCase(
	categories domain.CategoryRepository,
	products domain.ProductRepository,
	priceMode domain.PriceMode,
) CatalogUseCase {
	return &catalogUseCase{categories: categories, products: products, priceMode: priceMode}
}

// ListCategories devuelve las categorías activas.
func (uc *catalogUseCase) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return uc.categories.ListActive(ctx)
}

// ListProducts devuelve los productos vendibles, opcionalmente filtrados por
// categoría, junto con sus variantes activas.
func (uc *catalogUseCase) ListProducts(ctx context.Context, categoryID *string) ([]CatalogProduct, error) {
	products, err := uc.products.ListSellable(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	out := make([]CatalogProduct, 0, len(products))
	for i := range products {
		detail, err := uc.buildDetail(ctx, &products[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *detail)
	}
	return out, nil
}

// GetProduct devuelve el detalle público de un producto vendible. Si no existe,
// está inactivo o no tiene stock disponible, retorna domain.ErrNotFound.
func (uc *catalogUseCase) GetProduct(ctx context.Context, id string) (*CatalogProduct, error) {
	product, err := uc.products.GetSellableByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.buildDetail(ctx, product)
}

// buildDetail proyecta una variante al contrato público aplicando el modo de
// precios configurado.
func (uc *catalogUseCase) buildDetail(ctx context.Context, product *domain.Product) (*CatalogProduct, error) {
	variants, err := uc.products.ListActiveVariantsByProduct(ctx, product.ID)
	if err != nil {
		return nil, err
	}

	out := make([]CatalogVariant, 0, len(variants))
	for i := range variants {
		variant := &variants[i]
		attributes := variant.Attributes
		if attributes == nil {
			attributes = map[string]any{}
		}
		out = append(out, CatalogVariant{
			ID:            variant.ID,
			Name:          variant.Name,
			Price:         variant.PriceForMode(uc.priceMode),
			StockQuantity: variant.StockQuantity,
			Available:     variant.StockQuantity > 0,
			Attributes:    attributes,
		})
	}

	return &CatalogProduct{Product: product, Variants: out}, nil
}
