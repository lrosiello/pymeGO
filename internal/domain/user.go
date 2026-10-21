package domain

import (
	"context"
	"time"
)

// Role representa el rol de un usuario del comercio.
type Role string

// Roles soportados por el schema SQL.
const (
	RoleAdmin    Role = "admin"
	RoleStaff    Role = "staff"
	RoleCustomer Role = "customer"
)

// IsValid indica si el rol es uno de los permitidos.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleStaff, RoleCustomer:
		return true
	default:
		return false
	}
}

// User representa a una persona que accede al sistema: administrador, staff o
// cliente del comercio. El email es único a nivel global.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Role         Role
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserRepository es el puerto de persistencia para usuarios.
type UserRepository interface {
	// Create persiste un nuevo usuario y rellena su ID y timestamps.
	Create(ctx context.Context, user *User) error
	// GetByID recupera un usuario por su identificador.
	GetByID(ctx context.Context, id string) (*User, error)
	// GetByEmail recupera un usuario por su email (único global).
	GetByEmail(ctx context.Context, email string) (*User, error)
	// ExistsByEmail indica si ya existe un usuario con ese email.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
