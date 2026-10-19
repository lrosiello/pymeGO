package domain

import (
	"context"
	"time"
)

// Category agrupa productos y admite jerarquía mediante ParentID
// (autorreferencia opcional) para construir árboles de categorías.
type Category struct {
	ID          string
	ParentID    *string
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CategoryRepository es el puerto de persistencia para categorías.
//
// NOTA: su implementación PostgreSQL se integrará en product_repository.go y su
// lógica en el módulo de catálogo, posterior al módulo de autenticación. Se
// declara aquí para respetar el contrato de Clean Architecture.
type CategoryRepository interface {
	Create(ctx context.Context, category *Category) error
	GetByID(ctx context.Context, id string) (*Category, error)
	List(ctx context.Context) ([]Category, error)
	Update(ctx context.Context, category *Category) error
	Delete(ctx context.Context, id string) error
}
