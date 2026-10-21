// Package dto contiene los objetos de transferencia de datos expuestos por la
// API HTTP. Separa el contrato público del modelo de dominio y aloja las reglas
// de validación de entrada (go-playground/validator mediante tags "binding").
package dto

import (
	"time"

	"pymego/internal/domain"
	"pymego/internal/usecase"
)

// RegisterRequest es el cuerpo de POST /api/v1/auth/register.
type RegisterRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=150"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// ToInput traduce el DTO al input del caso de uso.
func (r RegisterRequest) ToInput() usecase.RegisterInput {
	return usecase.RegisterInput{
		FullName: r.FullName,
		Email:    r.Email,
		Password: r.Password,
	}
}

// LoginRequest es el cuerpo de POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required"`
}

// ToInput traduce el DTO al input del caso de uso.
func (r LoginRequest) ToInput() usecase.LoginInput {
	return usecase.LoginInput{
		Email:    r.Email,
		Password: r.Password,
	}
}

// UserResponse es la representación pública de un usuario (nunca incluye el hash).
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// NewUserResponse construye la representación pública desde la entidad de dominio.
func NewUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      string(user.Role),
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}
}

// AuthResponse es la respuesta de un registro o login exitoso.
type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}
