package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pymego/internal/domain"
)

// UserRepository es la implementación PostgreSQL de domain.UserRepository.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository construye el repositorio de usuarios.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Verificación en tiempo de compilación de que se cumple el puerto del dominio.
var _ domain.UserRepository = (*UserRepository)(nil)

const userColumns = `id, email, password_hash, full_name, role, is_active, created_at, updated_at`

// Create inserta un usuario y rellena ID y timestamps del struct recibido.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (email, password_hash, full_name, role, is_active)
	          VALUES ($1, $2, $3, $4, $5)
	          RETURNING id, created_at, updated_at`

	err := querier(ctx, r.pool).QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.FullName,
		string(user.Role),
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: el email '%s' ya está registrado", domain.ErrConflict, user.Email)
		}
		return fmt.Errorf("postgres: error al crear usuario: %w", err)
	}

	return nil
}

// GetByID recupera un usuario por su identificador.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	return r.scanOne(querier(ctx, r.pool).QueryRow(ctx, query, id))
}

// GetByEmail recupera un usuario por su email (único global).
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1`
	return r.scanOne(querier(ctx, r.pool).QueryRow(ctx, query, email))
}

// ExistsByEmail indica si ya existe un usuario con ese email.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	if err := querier(ctx, r.pool).QueryRow(ctx, query, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("postgres: error al verificar email: %w", err)
	}
	return exists, nil
}

// scanOne materializa una fila en un domain.User, traduciendo la ausencia de
// resultado a domain.ErrNotFound.
func (r *UserRepository) scanOne(row pgx.Row) (*domain.User, error) {
	var (
		user    domain.User
		rawRole string
	)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&rawRole,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: error al leer usuario: %w", err)
	}

	user.Role = domain.Role(rawRole)
	return &user, nil
}
