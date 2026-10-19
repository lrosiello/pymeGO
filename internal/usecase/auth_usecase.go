// Package usecase contiene la lógica de aplicación: orquesta entidades del
// dominio, repositorios y servicios externos (hashing, tokens) sin conocer
// detalles de transporte (HTTP) ni de persistencia concreta.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pymego/internal/domain"
)

// bcryptCost define el coste de hashing. 12 es un equilibrio razonable entre
// seguridad y latencia para un panel de administración.
const bcryptCost = 12

// TokenIssuer abstrae la emisión de tokens JWT para desacoplar el caso de uso
// de la implementación concreta (pkg/jwt).
type TokenIssuer interface {
	Generate(userID, role string) (token string, expiresAt time.Time, err error)
}

// AuthUseCase define las operaciones de autenticación disponibles.
type AuthUseCase interface {
	// Register crea al usuario administrador del comercio y devuelve un token.
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	// Login valida las credenciales de un usuario.
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
}

// RegisterInput son los datos necesarios para registrar al administrador.
type RegisterInput struct {
	FullName string
	Email    string
	Password string
}

// LoginInput son las credenciales de acceso.
type LoginInput struct {
	Email    string
	Password string
}

// AuthResult es el resultado de un registro o login exitoso.
type AuthResult struct {
	Token     string
	ExpiresAt time.Time
	User      *domain.User
}

// authUseCase implementa AuthUseCase.
type authUseCase struct {
	users  domain.UserRepository
	tokens TokenIssuer
}

// NewAuthUseCase construye el caso de uso de autenticación.
func NewAuthUseCase(users domain.UserRepository, tokens TokenIssuer) AuthUseCase {
	return &authUseCase{users: users, tokens: tokens}
}

// Register valida la entrada, garantiza la unicidad del email y crea al usuario
// administrador del comercio.
func (uc *authUseCase) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	email := normalizeEmail(input.Email)

	exists, err := uc.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: el email '%s' ya está registrado", domain.ErrConflict, email)
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: passwordHash,
		FullName:     strings.TrimSpace(input.FullName),
		Role:         domain.RoleAdmin,
		IsActive:     true,
	}

	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}

	token, expiresAt, err := uc.tokens.Generate(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}

	return &AuthResult{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

// Login verifica las credenciales y emite un token. Los mensajes de error son
// deliberadamente genéricos para no filtrar la existencia de usuarios.
func (uc *authUseCase) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	email := normalizeEmail(input.Email)

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	if !user.IsActive {
		return nil, fmt.Errorf("%w: el usuario está inactivo", domain.ErrUnauthorized)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	token, expiresAt, err := uc.tokens.Generate(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}

	return &AuthResult{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

// hashPassword genera el hash bcrypt de una contraseña.
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("usecase: no se pudo hashear la contraseña: %w", err)
	}
	return string(hash), nil
}

// normalizeEmail limpia y normaliza el email a minúsculas.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
