package jwt

import (
	"errors"
	"testing"
	"time"
)

const testSecret = "secreto-de-prueba-con-mas-de-32-bytes!!"

func TestGenerateAndParse(t *testing.T) {
	m := NewManager(testSecret, "catalogogo-test", time.Hour)

	token, expiresAt, err := m.Generate("user-1", "admin")
	if err != nil {
		t.Fatalf("Generate() error inesperado: %v", err)
	}
	if token == "" {
		t.Fatal("Generate() devolvió un token vacío")
	}
	if !expiresAt.After(time.Now()) {
		t.Errorf("expiresAt=%v debería ser futuro", expiresAt)
	}

	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error inesperado: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, se esperaba user-1", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, se esperaba admin", claims.Role)
	}
	if claims.Issuer != "catalogogo-test" {
		t.Errorf("Issuer = %q, se esperaba catalogogo-test", claims.Issuer)
	}
}

func TestParse_Expired(t *testing.T) {
	m := NewManager(testSecret, "catalogogo-test", -time.Hour)

	token, _, err := m.Generate("user-1", "admin")
	if err != nil {
		t.Fatalf("Generate() error inesperado: %v", err)
	}

	_, err = m.Parse(token)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Parse() = %v, se esperaba ErrExpiredToken", err)
	}
}

func TestParse_WrongSecret(t *testing.T) {
	issuer := NewManager(testSecret, "catalogogo-test", time.Hour)
	token, _, err := issuer.Generate("user-1", "admin")
	if err != nil {
		t.Fatalf("Generate() error inesperado: %v", err)
	}

	validator := NewManager("otro-secreto-completamente-distinto-1234", "catalogogo-test", time.Hour)
	if _, err := validator.Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse() = %v, se esperaba ErrInvalidToken", err)
	}
}

func TestParse_Malformed(t *testing.T) {
	m := NewManager(testSecret, "catalogogo-test", time.Hour)
	if _, err := m.Parse("esto.no.es.un.jwt"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse() = %v, se esperaba ErrInvalidToken", err)
	}
}
