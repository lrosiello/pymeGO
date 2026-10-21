// Package jwt encapsula la generación y validación de tokens JWT firmados con
// HMAC-SHA256 (HS256). Es agnóstico al dominio: solo conoce los claims que
// transporta.
package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Errores centinela para distinguir causas de fallo al validar un token.
var (
	ErrInvalidToken = errors.New("token inválido")
	ErrExpiredToken = errors.New("token expirado")
)

// Claims son los claims personalizados que viajan dentro del token. No hay
// contexto multi-tenant: solo la identidad del usuario y su rol.
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Manager emite y valida tokens JWT con una configuración fija.
type Manager struct {
	secret    []byte
	issuer    string
	expiresIn time.Duration
}

// NewManager crea un Manager. secret debe tener entropía suficiente (>= 32 bytes).
func NewManager(secret, issuer string, expiresIn time.Duration) *Manager {
	return &Manager{
		secret:    []byte(secret),
		issuer:    issuer,
		expiresIn: expiresIn,
	}
}

// Generate firma un token para el usuario indicado y retorna el string junto
// con el instante de expiración.
func (m *Manager) Generate(userID, role string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.expiresIn)

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("jwt: no se pudo firmar el token: %w", err)
	}

	return signed, expiresAt, nil
}

// Parse valida la firma y expiración del token y devuelve sus claims.
func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: método de firma inesperado: %v", t.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
