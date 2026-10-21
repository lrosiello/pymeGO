package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pymego/internal/domain"
)

func seededHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("no se pudo generar el hash de prueba: %v", err)
	}
	return string(hash)
}

func TestAuthRegister_Success(t *testing.T) {
	repo := newFakeUserRepo()
	tokens := &fakeTokenIssuer{token: "jwt-token", expiresAt: time.Now().Add(time.Hour)}
	uc := NewAuthUseCase(repo, tokens)

	res, err := uc.Register(context.Background(), RegisterInput{
		FullName: "Ana Gómez",
		Email:    "Ana@Example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("Register() error inesperado: %v", err)
	}

	if res.Token != "jwt-token" {
		t.Errorf("Token = %q, se esperaba jwt-token", res.Token)
	}
	if res.User.ID == "" {
		t.Error("el usuario debería tener ID")
	}
	if res.User.Email != "ana@example.com" {
		t.Errorf("Email = %q, se esperaba normalizado a minúsculas", res.User.Email)
	}
	if res.User.Role != domain.RoleAdmin {
		t.Errorf("Role = %q, se esperaba admin", res.User.Role)
	}
	if !res.User.IsActive {
		t.Error("el usuario debería crearse activo")
	}
	if res.User.PasswordHash == "secret123" {
		t.Error("la contraseña no debe almacenarse en texto plano")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(res.User.PasswordHash), []byte("secret123")); err != nil {
		t.Errorf("el hash almacenado no corresponde a la contraseña: %v", err)
	}
	if tokens.gotUserID != res.User.ID || tokens.gotRole != string(domain.RoleAdmin) {
		t.Errorf("el token se generó con claims inesperados: uid=%q role=%q", tokens.gotUserID, tokens.gotRole)
	}
}

func TestAuthRegister_DuplicateEmail(t *testing.T) {
	repo := newFakeUserRepo()
	repo.usersByEmail["ana@example.com"] = &domain.User{ID: "existing", Email: "ana@example.com"}
	uc := NewAuthUseCase(repo, &fakeTokenIssuer{token: "x"})

	_, err := uc.Register(context.Background(), RegisterInput{
		FullName: "Ana", Email: "ana@example.com", Password: "secret123",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Register() = %v, se esperaba ErrConflict", err)
	}
}

func TestAuthRegister_TokenError(t *testing.T) {
	repo := newFakeUserRepo()
	tokens := &fakeTokenIssuer{err: errors.New("fallo al firmar")}
	uc := NewAuthUseCase(repo, tokens)

	_, err := uc.Register(context.Background(), RegisterInput{
		FullName: "Ana", Email: "ana@example.com", Password: "secret123",
	})
	if err == nil {
		t.Fatal("se esperaba un error al fallar la emisión del token")
	}
}

func TestAuthLogin_Success(t *testing.T) {
	repo := newFakeUserRepo()
	user := &domain.User{
		ID: "user-1", Email: "ana@example.com", PasswordHash: seededHash(t, "secret123"),
		FullName: "Ana", Role: domain.RoleAdmin, IsActive: true,
	}
	repo.usersByEmail[user.Email] = user
	repo.usersByID[user.ID] = user

	tokens := &fakeTokenIssuer{token: "jwt-token", expiresAt: time.Now().Add(time.Hour)}
	uc := NewAuthUseCase(repo, tokens)

	res, err := uc.Login(context.Background(), LoginInput{Email: "ANA@example.com", Password: "secret123"})
	if err != nil {
		t.Fatalf("Login() error inesperado: %v", err)
	}
	if res.Token != "jwt-token" {
		t.Errorf("Token = %q, se esperaba jwt-token", res.Token)
	}
	if tokens.gotUserID != "user-1" || tokens.gotRole != "admin" {
		t.Errorf("claims inesperados: uid=%q role=%q", tokens.gotUserID, tokens.gotRole)
	}
}

func TestAuthLogin_WrongPassword(t *testing.T) {
	repo := newFakeUserRepo()
	user := &domain.User{
		ID: "user-1", Email: "ana@example.com", PasswordHash: seededHash(t, "secret123"),
		Role: domain.RoleAdmin, IsActive: true,
	}
	repo.usersByEmail[user.Email] = user

	uc := NewAuthUseCase(repo, &fakeTokenIssuer{token: "x"})
	if _, err := uc.Login(context.Background(), LoginInput{Email: "ana@example.com", Password: "incorrecta"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Login() = %v, se esperaba ErrUnauthorized", err)
	}
}

func TestAuthLogin_UserNotFound(t *testing.T) {
	uc := NewAuthUseCase(newFakeUserRepo(), &fakeTokenIssuer{token: "x"})
	if _, err := uc.Login(context.Background(), LoginInput{Email: "nadie@example.com", Password: "x"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Login() = %v, se esperaba ErrUnauthorized", err)
	}
}

func TestAuthLogin_Inactive(t *testing.T) {
	repo := newFakeUserRepo()
	user := &domain.User{
		ID: "user-1", Email: "ana@example.com", PasswordHash: seededHash(t, "secret123"),
		Role: domain.RoleAdmin, IsActive: false,
	}
	repo.usersByEmail[user.Email] = user

	uc := NewAuthUseCase(repo, &fakeTokenIssuer{token: "x"})
	if _, err := uc.Login(context.Background(), LoginInput{Email: "ana@example.com", Password: "secret123"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Login() = %v, se esperaba ErrUnauthorized", err)
	}
}
