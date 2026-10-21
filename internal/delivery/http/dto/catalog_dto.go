package dto

import (
	"pymego/internal/domain"
	"pymego/internal/usecase"
)

// CatalogCategoryResponse es la representación pública de una categoría activa.
type CatalogCategoryResponse struct {
	ID          string  `json:"id"`
	ParentID    *string `json:"parent_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
}

// NewCatalogCategoryResponse construye la respuesta desde la entidad de dominio.
func NewCatalogCategoryResponse(category *domain.Category) CatalogCategoryResponse {
	return CatalogCategoryResponse{
		ID:          category.ID,
		ParentID:    category.ParentID,
		Name:        category.Name,
		Description: category.Description,
	}
}

// CatalogVariantResponse es la representación pública de una variante. Solo
// expone el precio efectivo según CATALOG_PRICE_MODE.
type CatalogVariantResponse struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Price         float64        `json:"price"`
	StockQuantity int            `json:"stock_quantity"`
	Available     bool           `json:"available"`
	Attributes    map[string]any `json:"attributes"`
}

// CatalogProductResponse es la representación pública de un producto vendible.
type CatalogProductResponse struct {
	ID          string                   `json:"id"`
	CategoryID  *string                  `json:"category_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Images      []string                 `json:"images"`
	Variants    []CatalogVariantResponse `json:"variants"`
}

// NewCatalogProductResponse construye la respuesta pública (producto +
// variantes) desde la proyección del caso de uso.
func NewCatalogProductResponse(detail usecase.CatalogProduct) CatalogProductResponse {
	images := detail.Product.Images
	if images == nil {
		images = []string{}
	}

	variants := make([]CatalogVariantResponse, 0, len(detail.Variants))
	for _, variant := range detail.Variants {
		attributes := variant.Attributes
		if attributes == nil {
			attributes = map[string]any{}
		}
		variants = append(variants, CatalogVariantResponse{
			ID:            variant.ID,
			Name:          variant.Name,
			Price:         variant.Price,
			StockQuantity: variant.StockQuantity,
			Available:     variant.Available,
			Attributes:    attributes,
		})
	}

	return CatalogProductResponse{
		ID:          detail.Product.ID,
		CategoryID:  detail.Product.CategoryID,
		Name:        detail.Product.Name,
		Description: detail.Product.Description,
		Images:      images,
		Variants:    variants,
	}
}
