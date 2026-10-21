package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pymego/internal/domain"
)

// Este archivo agrupa la persistencia del módulo de catálogo, con SQL explícito:
//   - CategoryRepository: categorías (domain.CategoryRepository)
//   - ProductRepository:  productos y variantes con stock atómico
//     (domain.ProductRepository)
//
// Todas las operaciones respetan la transacción activa del contexto (si existe)
// a través del helper querier, lo que permite composiciones atómicas desde la
// capa de casos de uso.

const (
	productColumns  = `id, category_id, name, description, images, is_active, created_at, updated_at`
	variantColumns  = `id, product_id, sku, name, price, b2b_price, stock_quantity, attributes, is_active, created_at, updated_at`
	categoryColumns = `id, parent_id, name, description, is_active, created_at, updated_at`
)

// ---------------------------------------------------------------------------
// Categorías
// ---------------------------------------------------------------------------

// CategoryRepository implementa domain.CategoryRepository.
type CategoryRepository struct {
	pool *pgxpool.Pool
}

// NewCategoryRepository construye el repositorio de categorías.
func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

var _ domain.CategoryRepository = (*CategoryRepository)(nil)

// Create inserta una categoría y rellena ID y timestamps.
func (r *CategoryRepository) Create(ctx context.Context, category *domain.Category) error {
	query := `INSERT INTO categories (parent_id, name, description, is_active)
	          VALUES ($1, $2, $3, $4)
	          RETURNING id, created_at, updated_at`

	err := querier(ctx, r.pool).QueryRow(ctx, query,
		category.ParentID,
		category.Name,
		category.Description,
		category.IsActive,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: error al crear categoría: %w", err)
	}

	return nil
}

// GetByID recupera una categoría por su identificador.
func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	query := `SELECT ` + categoryColumns + ` FROM categories WHERE id = $1`
	return scanCategory(querier(ctx, r.pool).QueryRow(ctx, query, id))
}

// List devuelve todas las categorías ordenadas por nombre.
func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	query := `SELECT ` + categoryColumns + ` FROM categories ORDER BY name ASC`

	rows, err := querier(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: error al listar categorías: %w", err)
	}
	defer rows.Close()

	categories := make([]domain.Category, 0)
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, *category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: error al iterar categorías: %w", err)
	}

	return categories, nil
}

// ListActive devuelve solo las categorías activas (vitrina pública).
func (r *CategoryRepository) ListActive(ctx context.Context) ([]domain.Category, error) {
	query := `SELECT ` + categoryColumns + ` FROM categories WHERE is_active = TRUE ORDER BY name ASC`

	rows, err := querier(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: error al listar categorías activas: %w", err)
	}
	defer rows.Close()

	categories := make([]domain.Category, 0)
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, *category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: error al iterar categorías activas: %w", err)
	}

	return categories, nil
}

// Update actualiza los campos mutables de una categoría.
func (r *CategoryRepository) Update(ctx context.Context, category *domain.Category) error {
	query := `UPDATE categories
	          SET parent_id = $2, name = $3, description = $4, is_active = $5, updated_at = NOW()
	          WHERE id = $1
	          RETURNING updated_at`

	err := querier(ctx, r.pool).QueryRow(ctx, query,
		category.ID,
		category.ParentID,
		category.Name,
		category.Description,
		category.IsActive,
	).Scan(&category.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("postgres: error al actualizar categoría: %w", err)
	}

	return nil
}

// Delete elimina una categoría por su identificador.
func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	return deleteByID(ctx, querier(ctx, r.pool), "categories", id)
}

// ---------------------------------------------------------------------------
// Productos
// ---------------------------------------------------------------------------

// ProductRepository implementa domain.ProductRepository.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository construye el repositorio de productos y variantes.
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

var _ domain.ProductRepository = (*ProductRepository)(nil)

// Create inserta un producto y rellena ID y timestamps.
func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	query := `INSERT INTO products (category_id, name, description, images, is_active)
	          VALUES ($1, $2, $3, $4, $5)
	          RETURNING id, created_at, updated_at`

	err := querier(ctx, r.pool).QueryRow(ctx, query,
		product.CategoryID,
		product.Name,
		product.Description,
		product.Images,
		product.IsActive,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: error al crear producto: %w", err)
	}

	return nil
}

// GetByID recupera un producto por su identificador.
func (r *ProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products WHERE id = $1`
	return scanProduct(querier(ctx, r.pool).QueryRow(ctx, query, id))
}

// List devuelve todos los productos ordenados por fecha de creación.
func (r *ProductRepository) List(ctx context.Context) ([]domain.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products ORDER BY created_at DESC`

	rows, err := querier(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: error al listar productos: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, *product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: error al iterar productos: %w", err)
	}

	return products, nil
}

// Update actualiza los campos mutables de un producto.
func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) error {
	query := `UPDATE products
	          SET category_id = $2, name = $3, description = $4, images = $5, is_active = $6, updated_at = NOW()
	          WHERE id = $1
	          RETURNING updated_at`

	err := querier(ctx, r.pool).QueryRow(ctx, query,
		product.ID,
		product.CategoryID,
		product.Name,
		product.Description,
		product.Images,
		product.IsActive,
	).Scan(&product.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("postgres: error al actualizar producto: %w", err)
	}

	return nil
}

// Delete elimina un producto (y sus variantes en cascada) por su ID.
func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	return deleteByID(ctx, querier(ctx, r.pool), "products", id)
}

// ---------------------------------------------------------------------------
// Variantes de producto
// ---------------------------------------------------------------------------

// CreateVariant inserta una variante y rellena ID y timestamps.
func (r *ProductRepository) CreateVariant(ctx context.Context, variant *domain.ProductVariant) error {
	attributes, err := marshalJSONB(variant.Attributes)
	if err != nil {
		return err
	}

	query := `INSERT INTO product_variants
	          (product_id, sku, name, price, b2b_price, stock_quantity, attributes, is_active)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	          RETURNING id, created_at, updated_at`

	err = querier(ctx, r.pool).QueryRow(ctx, query,
		variant.ProductID,
		variant.SKU,
		variant.Name,
		variant.Price,
		variant.B2BPrice,
		variant.StockQuantity,
		attributes,
		variant.IsActive,
	).Scan(&variant.ID, &variant.CreatedAt, &variant.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: ya existe una variante con el SKU '%s'", domain.ErrConflict, variant.SKU)
		}
		return fmt.Errorf("postgres: error al crear variante: %w", err)
	}

	return nil
}

// GetVariantByID recupera una variante por su identificador.
func (r *ProductRepository) GetVariantByID(ctx context.Context, id string) (*domain.ProductVariant, error) {
	query := `SELECT ` + variantColumns + ` FROM product_variants WHERE id = $1`
	return scanVariant(querier(ctx, r.pool).QueryRow(ctx, query, id))
}

// ListVariantsByProduct devuelve las variantes de un producto.
func (r *ProductRepository) ListVariantsByProduct(ctx context.Context, productID string) ([]domain.ProductVariant, error) {
	query := `SELECT ` + variantColumns + ` FROM product_variants WHERE product_id = $1 ORDER BY created_at ASC`

	rows, err := querier(ctx, r.pool).Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("postgres: error al listar variantes: %w", err)
	}
	defer rows.Close()

	variants := make([]domain.ProductVariant, 0)
	for rows.Next() {
		variant, err := scanVariant(rows)
		if err != nil {
			return nil, err
		}
		variants = append(variants, *variant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: error al iterar variantes: %w", err)
	}

	return variants, nil
}

// ListSellable devuelve los productos visibles en la vitrina pública: activos y
// con al menos una variante activa con stock disponible. Si categoryID no es
// nil, restringe el resultado a esa categoría.
func (r *ProductRepository) ListSellable(ctx context.Context, categoryID *string) ([]domain.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products
	          WHERE is_active = TRUE
	            AND ($1::uuid IS NULL OR category_id = $1::uuid)
	            AND EXISTS (
	                SELECT 1 FROM product_variants v
	                WHERE v.product_id = products.id
	                  AND v.is_active = TRUE
	                  AND v.stock_quantity > 0
	            )
	          ORDER BY created_at DESC`

	rows, err := querier(ctx, r.pool).Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("postgres: error al listar productos vendibles: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, *product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: error al iterar productos vendibles: %w", err)
	}

	return products, nil
}

// GetSellableByID devuelve un producto visible en la vitrina pública o
// domain.ErrNotFound si está inactivo, no existe o no tiene stock disponible.
func (r *ProductRepository) GetSellableByID(ctx context.Context, id string) (*domain.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products
	          WHERE id = $1
	            AND is_active = TRUE
	            AND EXISTS (
	                SELECT 1 FROM product_variants v
	                WHERE v.product_id = products.id
	                  AND v.is_active = TRUE
	                  AND v.stock_quantity > 0
	            )`

	return scanProduct(querier(ctx, r.pool).QueryRow(ctx, query, id))
}

// ListActiveVariantsByProduct devuelve las variantes activas de un producto.
func (r *ProductRepository) ListActiveVariantsByProduct(ctx context.Context, productID string) ([]domain.ProductVariant, error) {
	query := `SELECT ` + variantColumns + ` FROM product_variants
	          WHERE product_id = $1 AND is_active = TRUE
	          ORDER BY created_at ASC`

	rows, err := querier(ctx, r.pool).Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("postgres: error al listar variantes activas: %w", err)
	}
	defer rows.Close()

	variants := make([]domain.ProductVariant, 0)
	for rows.Next() {
		variant, err := scanVariant(rows)
		if err != nil {
			return nil, err
		}
		variants = append(variants, *variant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: error al iterar variantes activas: %w", err)
	}

	return variants, nil
}

// UpdateVariant actualiza los campos mutables de una variante. El stock no se
// modifica aquí: se ajusta atómicamente mediante AdjustStockQuantity.
func (r *ProductRepository) UpdateVariant(ctx context.Context, variant *domain.ProductVariant) error {
	attributes, err := marshalJSONB(variant.Attributes)
	if err != nil {
		return err
	}

	query := `UPDATE product_variants
	          SET name = $2, sku = $3, price = $4, b2b_price = $5, attributes = $6, is_active = $7, updated_at = NOW()
	          WHERE id = $1
	          RETURNING updated_at`

	err = querier(ctx, r.pool).QueryRow(ctx, query,
		variant.ID,
		variant.Name,
		variant.SKU,
		variant.Price,
		variant.B2BPrice,
		attributes,
		variant.IsActive,
	).Scan(&variant.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: ya existe una variante con el SKU '%s'", domain.ErrConflict, variant.SKU)
		}
		return fmt.Errorf("postgres: error al actualizar variante: %w", err)
	}

	return nil
}

// DeleteVariant elimina una variante por su identificador.
func (r *ProductRepository) DeleteVariant(ctx context.Context, id string) error {
	return deleteByID(ctx, querier(ctx, r.pool), "product_variants", id)
}

// AdjustStockQuantity suma delta al stock de forma atómica. La condición
// "stock_quantity + delta >= 0" se evalúa dentro de la misma sentencia SQL,
// evitando condiciones de carrera entre lecturas y escrituras concurrentes.
func (r *ProductRepository) AdjustStockQuantity(ctx context.Context, id string, delta int) (*domain.ProductVariant, error) {
	query := `UPDATE product_variants
	          SET stock_quantity = stock_quantity + $2, updated_at = NOW()
	          WHERE id = $1 AND stock_quantity + $2 >= 0
	          RETURNING ` + variantColumns

	variant, err := scanVariant(querier(ctx, r.pool).QueryRow(ctx, query, id, delta))
	if err == nil {
		return variant, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	// No se actualizó ninguna fila: distinguimos "no existe" de "stock
	// insuficiente" consultando la variante actual.
	existing, getErr := r.GetVariantByID(ctx, id)
	if getErr != nil {
		return nil, getErr
	}
	return nil, domain.NewValidationError(
		"stock",
		fmt.Sprintf("stock insuficiente: disponible %d, solicitado %d", existing.StockQuantity, -delta),
	)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// scanCategory materializa una fila en domain.Category.
func scanCategory(row pgx.Row) (*domain.Category, error) {
	var category domain.Category

	err := row.Scan(
		&category.ID,
		&category.ParentID,
		&category.Name,
		&category.Description,
		&category.IsActive,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: error al leer categoría: %w", err)
	}

	return &category, nil
}

// scanProduct materializa una fila en domain.Product.
func scanProduct(row pgx.Row) (*domain.Product, error) {
	var product domain.Product

	err := row.Scan(
		&product.ID,
		&product.CategoryID,
		&product.Name,
		&product.Description,
		&product.Images,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: error al leer producto: %w", err)
	}

	return &product, nil
}

// scanVariant materializa una fila en domain.ProductVariant.
func scanVariant(row pgx.Row) (*domain.ProductVariant, error) {
	var (
		variant       domain.ProductVariant
		rawAttributes []byte
	)

	err := row.Scan(
		&variant.ID,
		&variant.ProductID,
		&variant.SKU,
		&variant.Name,
		&variant.Price,
		&variant.B2BPrice,
		&variant.StockQuantity,
		&rawAttributes,
		&variant.IsActive,
		&variant.CreatedAt,
		&variant.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: error al leer variante: %w", err)
	}

	if err := unmarshalJSONB(rawAttributes, &variant.Attributes); err != nil {
		return nil, err
	}

	return &variant, nil
}

// deleteByID elimina una fila por id y traduce "0 filas" a domain.ErrNotFound.
// El nombre de tabla lo controla el código interno (nunca entrada del usuario).
func deleteByID(ctx context.Context, db DBTX, table, id string) error {
	tag, err := db.Exec(ctx, "DELETE FROM "+table+" WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("postgres: error al eliminar en %s: %w", table, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// marshalJSONB serializa un mapa a JSON para columnas JSONB, devolviendo un
// objeto vacío cuando el mapa es nil.
func marshalJSONB(value map[string]any) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("postgres: no se pudo serializar JSONB: %w", err)
	}
	return raw, nil
}

// unmarshalJSONB deserializa JSONB en el destino indicado, tolerando nulos.
func unmarshalJSONB(raw []byte, dest *map[string]any) error {
	if len(raw) == 0 {
		*dest = map[string]any{}
		return nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("postgres: no se pudo deserializar JSONB: %w", err)
	}
	return nil
}
