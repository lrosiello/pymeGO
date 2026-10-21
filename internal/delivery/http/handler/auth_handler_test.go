package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"pymego/internal/domain"
	"pymego/internal/usecase"
)

func authResult() *usecase.AuthResult {
	return &usecase.AuthResult{
		Token:     "jwt-token",
		ExpiresAt: time.Now().Add(time.Hour),
		User: &domain.User{
			ID: "user-1", Email: "ana@example.com", FullName: "Ana", Role: domain.RoleAdmin, IsActive: true,
		},
	}
}

func TestAuthHandler_Register_Created(t *testing.T) {
	uc := &fakeAuthUC{registerFn: func(_ context.Context, in usecase.RegisterInput) (*usecase.AuthResult, error) {
		if in.Email != "ana@example.com" {
			t.Errorf("email no propagado al caso de uso: %q", in.Email)
		}
		return authResult(), nil
	}}
	h := NewAuthHandler(uc, testLogger())

	w := performRequest(t, h.Register, http.MethodPost, "/api/v1/auth/register",
		`{"full_name":"Ana","email":"ana@example.com","password":"secret123"}`, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201; body=%s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if body["token"] != "jwt-token" {
		t.Errorf("token = %v, se esperaba jwt-token", body["token"])
	}
	user, _ := body["user"].(map[string]any)
	if user["email"] != "ana@example.com" {
		t.Errorf("user.email = %v", user["email"])
	}
	if _, ok := user["password_hash"]; ok {
		t.Error("la respuesta no debe exponer password_hash")
	}
}

func TestAuthHandler_Register_BadRequest(t *testing.T) {
	h := NewAuthHandler(&fakeAuthUC{}, testLogger())

	w := performRequest(t, h.Register, http.MethodPost, "/api/v1/auth/register",
		`{"full_name":"Ana","email":"no-es-email","password":"secret123"}`, nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400; body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Register_Conflict(t *testing.T) {
	uc := &fakeAuthUC{registerFn: func(context.Context, usecase.RegisterInput) (*usecase.AuthResult, error) {
		return nil, domain.ErrConflict
	}}
	h := NewAuthHandler(uc, testLogger())

	w := performRequest(t, h.Register, http.MethodPost, "/api/v1/auth/register",
		`{"full_name":"Ana","email":"ana@example.com","password":"secret123"}`, nil)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, se esperaba 409; body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Login_OK(t *testing.T) {
	h := NewAuthHandler(&fakeAuthUC{loginFn: func(context.Context, usecase.LoginInput) (*usecase.AuthResult, error) {
		return authResult(), nil
	}}, testLogger())

	w := performRequest(t, h.Login, http.MethodPost, "/api/v1/auth/login",
		`{"email":"ana@example.com","password":"secret123"}`, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200; body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Login_Unauthorized(t *testing.T) {
	h := NewAuthHandler(&fakeAuthUC{loginFn: func(context.Context, usecase.LoginInput) (*usecase.AuthResult, error) {
		return nil, domain.ErrUnauthorized
	}}, testLogger())

	w := performRequest(t, h.Login, http.MethodPost, "/api/v1/auth/login",
		`{"email":"ana@example.com","password":"mala"}`, nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401; body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Login_UnexpectedError(t *testing.T) {
	h := NewAuthHandler(&fakeAuthUC{loginFn: func(context.Context, usecase.LoginInput) (*usecase.AuthResult, error) {
		return nil, errors.New("boom")
	}}, testLogger())

	w := performRequest(t, h.Login, http.MethodPost, "/api/v1/auth/login",
		`{"email":"ana@example.com","password":"secret123"}`, nil)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, se esperaba 500; body=%s", w.Code, w.Body.String())
	}
}
