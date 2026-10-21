package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"pymego/internal/domain"
)

// ProductUseCase define las operaciones del módulo de catálogo: categorías,
// productos y variantes con stock atómico.
type ProductUseCase interface {
	// Categorías.
	CreateCategory(ctx context.Context, input CreateCategoryInput) (*domain.Category, error)
	ListCategories(ctx context.Context) ([]domain.Category, error)
	UpdateCategory(ctx context.Context, id string, input UpdateCategoryInput) (*domain.Category, error)
	DeleteCategory(ctx context.Context, id string) error

	// Productos.
	CreateProduct(ctx context.Context, input CreateProductInput) (*ProductDetail, error)
	ListProducts(ctx context.Context) ([]ProductDetail, error)
	GetProduct(ctx context.Context, id string) (*ProductDetail, error)
	UpdateProduct(ctx context.Context, id string, input UpdateProductInput) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id string) error

	// Variantes.
	ListVariants(ctx context.Context, productID string) ([]domain.ProductVariant, error)
	AddVariant(ctx context.Context, productID string, input CreateVariantInput) (*domain.ProductVariant, error)
	UpdateVariant(ctx context.Context, variantID string, input UpdateVariantInput) (*domain.ProductVariant, error)
	DeleteVariant(ctx context.Context, variantID string) error
	AdjustStock(ctx context.Context, variantID string, delta int) (*domain.ProductVariant, error)
}

// CreateCategoryInput son los datos para crear una categoría.
type CreateCategoryInput struct {
	ParentID    *string
	Name        string
	Description string
}

// UpdateCategoryInput son los campos mutables de una categoría.
type UpdateCategoryInput struct {
	ParentID    *string
	Name        string
	Description string
	IsActive    bool
}

// CreateProductInput son los datos para crear un producto, opcionalmente con
// sus variantes (alta atómica).
type CreateProductInput struct {
	CategoryID  *string
	Name        string
	Description string
	Images      []string
	IsActive    bool
	Variants    []CreateVariantInput
}

// UpdateProductInput son los campos mutables de un producto.
type UpdateProductInput struct {
	CategoryID  *string
	Name        string
	Description string
	Images      []string
	IsActive    bool
}

// CreateVariantInput son los datos para crear una variante.
type CreateVariantInput struct {
	SKU           string
	Name          string
	Price         float64
	B2BPrice      *float64
	StockQuantity int
	Attributes    map[string]any
}

// UpdateVariantInput son los campos mutables de una variante (el stock se
// gestiona aparte mediante AdjustStock).
type UpdateVariantInput struct {
	SKU        string
	Name       string
	Price      float64
	B2BPrice   *float64
	Attributes map[string]any
	IsActive   bool
}

// ProductDetail agrupa un producto con sus variantes para las respuestas de
// lectura.
type ProductDetail struct {
	Product  *domain.Product
	Variants []domain.ProductVariant
}

// productUseCase implementa ProductUseCase.
type productUseCase struct {
	categories domain.CategoryRepository
	products   domain.ProductRepository
	tx         domain.TxManager
}

// NewProductUseCase construye el caso de uso de catálogo.
func NewProductUseCase(
	categories domain.CategoryRepository,
	products domain.ProductRepository,
	tx domain.TxManager,
) ProductUseCase {
	return &productUseCase{categories: categories, products: products, tx: tx}
}

// --- Categorías -------------------------------------------------------------

// CreateCategory valida y crea una categoría. Si se indica ParentID, verifica
// que la categoría padre exista.
func (uc *productUseCase) CreateCategory(ctx context.Context, input CreateCategoryInput) (*domain.Category, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.NewValidationError("name", "es obligatorio")
	}
	if err := uc.ensureCategoryExists(ctx, input.ParentID); err != nil {
		return nil, err
	}

	category := &domain.Category{
		ParentID:    input.ParentID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		IsActive:    true,
	}
	if err := uc.categories.Create(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

// ListCategories devuelve todas las categorías.
func (uc *productUseCase) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return uc.categories.List(ctx)
}

// UpdateCategory actualiza una categoría existente.
func (uc *productUseCase) UpdateCategory(ctx context.Context, id string, input UpdateCategoryInput) (*domain.Category, error) {
	category, err := uc.categories.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.NewValidationError("name", "es obligatorio")
	}
	if err := uc.ensureCategoryExists(ctx, input.ParentID); err != nil {
		return nil, err
	}
	if input.ParentID != nil && *input.ParentID == id {
		return nil, domain.NewValidationError("parent_id", "una categoría no puede ser su propio padre")
	}

	category.ParentID = input.ParentID
	category.Name = name
	category.Description = strings.TrimSpace(input.Description)
	category.IsActive = input.IsActive

	if err := uc.categories.Update(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

// DeleteCategory elimina una categoría por su identificador.
func (uc *productUseCase) DeleteCategory(ctx context.Context, id string) error {
	return uc.categories.Delete(ctx, id)
}

// --- Productos --------------------------------------------------------------

// CreateProduct valida y crea un producto junto con sus variantes de forma
// atómica: si alguna variante falla, no queda ningún registro parcial.
func (uc *productUseCase) CreateProduct(ctx context.Context, input CreateProductInput) (*ProductDetail, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.NewValidationError("name", "es obligatorio")
	}
	if err := uc.ensureCategoryExists(ctx, input.CategoryID); err != nil {
		return nil, err
	}

	product := &domain.Product{
		CategoryID:  input.CategoryID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Images:      input.Images,
		IsActive:    input.IsActive,
	}

	variants := make([]domain.ProductVariant, 0, len(input.Variants))

	err := uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.products.Create(txCtx, product); err != nil {
			return err
		}
		for i, in := range input.Variants {
			variant, err := buildVariant(product.ID, in)
			if err != nil {
				return fmt.Errorf("variante[%d]: %w", i, err)
			}
			if err := uc.products.CreateVariant(txCtx, variant); err != nil {
				return err
			}
			variants = append(variants, *variant)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &ProductDetail{Product: product, Variants: variants}, nil
}

// ListProducts devuelve todos los productos con sus variantes.
func (uc *productUseCase) ListProducts(ctx context.Context) ([]ProductDetail, error) {
	products, err := uc.products.List(ctx)
	if err != nil {
		return nil, err
	}

	details := make([]ProductDetail, 0, len(products))
	for i := range products {
		variants, err := uc.products.ListVariantsByProduct(ctx, products[i].ID)
		if err != nil {
			return nil, err
		}
		details = append(details, ProductDetail{Product: &products[i], Variants: variants})
	}
	return details, nil
}

// GetProduct devuelve un producto con sus variantes.
func (uc *productUseCase) GetProduct(ctx context.Context, id string) (*ProductDetail, error) {
	product, err := uc.products.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	variants, err := uc.products.ListVariantsByProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	return &ProductDetail{Product: product, Variants: variants}, nil
}

// UpdateProduct actualiza los campos mutables de un producto.
func (uc *productUseCase) UpdateProduct(ctx context.Context, id string, input UpdateProductInput) (*domain.Product, error) {
	product, err := uc.products.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.NewValidationError("name", "es obligatorio")
	}
	if err := uc.ensureCategoryExists(ctx, input.CategoryID); err != nil {
		return nil, err
	}

	product.CategoryID = input.CategoryID
	product.Name = name
	product.Description = strings.TrimSpace(input.Description)
	product.Images = input.Images
	product.IsActive = input.IsActive

	if err := uc.products.Update(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

// DeleteProduct elimina un producto y sus variantes en cascada.
func (uc *productUseCase) DeleteProduct(ctx context.Context, id string) error {
	return uc.products.Delete(ctx, id)
}

// --- Variantes --------------------------------------------------------------

// ListVariants devuelve las variantes de un producto, validando que exista.
func (uc *productUseCase) ListVariants(ctx context.Context, productID string) ([]domain.ProductVariant, error) {
	if _, err := uc.products.GetByID(ctx, productID); err != nil {
		return nil, err
	}
	return uc.products.ListVariantsByProduct(ctx, productID)
}

// AddVariant agrega una variante a un producto existente.
func (uc *productUseCase) AddVariant(ctx context.Context, productID string, input CreateVariantInput) (*domain.ProductVariant, error) {
	if _, err := uc.products.GetByID(ctx, productID); err != nil {
		return nil, err
	}

	variant, err := buildVariant(productID, input)
	if err != nil {
		return nil, err
	}
	if err := uc.products.CreateVariant(ctx, variant); err != nil {
		return nil, err
	}
	return variant, nil
}

// UpdateVariant actualiza los campos mutables de una variante (sin tocar stock).
func (uc *productUseCase) UpdateVariant(ctx context.Context, variantID string, input UpdateVariantInput) (*domain.ProductVariant, error) {
	variant, err := uc.products.GetVariantByID(ctx, variantID)
	if err != nil {
		return nil, err
	}

	sku := strings.TrimSpace(input.SKU)
	if sku == "" {
		return nil, domain.NewValidationError("sku", "es obligatorio")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.NewValidationError("name", "es obligatorio")
	}
	if err := validatePrice(input.Price, input.B2BPrice); err != nil {
		return nil, err
	}

	variant.SKU = sku
	variant.Name = name
	variant.Price = input.Price
	variant.B2BPrice = input.B2BPrice
	variant.Attributes = input.Attributes
	variant.IsActive = input.IsActive

	if err := uc.products.UpdateVariant(ctx, variant); err != nil {
		return nil, err
	}
	return variant, nil
}

// DeleteVariant elimina una variante por su identificador.
func (uc *productUseCase) DeleteVariant(ctx context.Context, variantID string) error {
	return uc.products.DeleteVariant(ctx, variantID)
}

// AdjustStock ajusta el stock de una variante de forma atómica. delta puede ser
// positivo (ingreso) o negativo (venta/reserva).
func (uc *productUseCase) AdjustStock(ctx context.Context, variantID string, delta int) (*domain.ProductVariant, error) {
	if delta == 0 {
		return nil, domain.NewValidationError("delta", "no puede ser cero")
	}
	return uc.products.AdjustStockQuantity(ctx, variantID, delta)
}

// --- Helpers ----------------------------------------------------------------

// ensureCategoryExists valida que, si se indica una categoría, esta exista.
func (uc *productUseCase) ensureCategoryExists(ctx context.Context, categoryID *string) error {
	if categoryID == nil || strings.TrimSpace(*categoryID) == "" {
		return nil
	}
	if _, err := uc.categories.GetByID(ctx, *categoryID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewValidationError("category_id", "la categoría indicada no existe")
		}
		return err
	}
	return nil
}

// buildVariant valida y construye una entidad de variante lista para persistir.
func buildVariant(productID string, input CreateVariantInput) (*domain.ProductVariant, error) {
	sku := strings.TrimSpace(input.SKU)
	if sku == "" {
		return nil, domain.NewValidationError("sku", "es obligatorio")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.NewValidationError("name", "es obligatorio")
	}
	if err := validatePrice(input.Price, input.B2BPrice); err != nil {
		return nil, err
	}
	if input.StockQuantity < 0 {
		return nil, domain.NewValidationError("stock_quantity", "no puede ser negativo")
	}

	attributes := input.Attributes
	if attributes == nil {
		attributes = map[string]any{}
	}

	return &domain.ProductVariant{
		ProductID:     productID,
		SKU:           sku,
		Name:          name,
		Price:         input.Price,
		B2BPrice:      input.B2BPrice,
		StockQuantity: input.StockQuantity,
		Attributes:    attributes,
		IsActive:      true,
	}, nil
}

// validatePrice comprueba que los precios sean no negativos.
func validatePrice(price float64, b2bPrice *float64) error {
	if price < 0 {
		return domain.NewValidationError("price", "no puede ser negativo")
	}
	if b2bPrice != nil && *b2bPrice < 0 {
		return domain.NewValidationError("b2b_price", "no puede ser negativo")
	}
	return nil
}
