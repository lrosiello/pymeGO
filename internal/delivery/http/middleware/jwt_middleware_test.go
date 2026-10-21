package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pymego/pkg/jwt"
)

func newRouter(t *testing.T) (*gin.Engine, *jwt.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	manager := jwt.NewManager("secreto-de-prueba-con-mas-de-32-bytes!!", "catalogogo-test", time.Hour)

	r := gin.New()
	r.Use(JWTMiddleware(manager))
	r.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": GetUserID(c), "role": GetRole(c)})
	})
	r.GET("/admin", RequireRole("admin", "staff"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	return r, manager
}

func do(r *gin.Engine, path, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestJWTMiddleware_MissingHeader(t *testing.T) {
	r, _ := newRouter(t)
	if w := do(r, "/me", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401", w.Code)
	}
}

func TestJWTMiddleware_InvalidScheme(t *testing.T) {
	r, _ := newRouter(t)
	if w := do(r, "/me", "Basic abc"); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401", w.Code)
	}
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	r, _ := newRouter(t)
	if w := do(r, "/me", "Bearer no-es-un-jwt"); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401", w.Code)
	}
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	r, manager := newRouter(t)
	token, _, err := manager.Generate("user-1", "admin")
	if err != nil {
		t.Fatalf("Generate() error inesperado: %v", err)
	}

	w := do(r, "/me", "Bearer "+token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200; body=%s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if body["user_id"] != "user-1" || body["role"] != "admin" {
		t.Errorf("identidad inesperada en contexto: %+v", body)
	}
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expired := jwt.NewManager("secreto-de-prueba-con-mas-de-32-bytes!!", "catalogogo-test", -time.Hour)
	token, _, err := expired.Generate("user-1", "admin")
	if err != nil {
		t.Fatalf("Generate() error inesperado: %v", err)
	}

	r := gin.New()
	r.Use(JWTMiddleware(expired))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusOK) })

	if w := do(r, "/me", "Bearer "+token); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401", w.Code)
	}
}

func TestRequireRole(t *testing.T) {
	r, manager := newRouter(t)

	adminToken, _, err := manager.Generate("user-1", "admin")
	if err != nil {
		t.Fatalf("Generate() error inesperado: %v", err)
	}
	customerToken, _, err := manager.Generate("user-2", "customer")
	if err != nil {
		t.Fatalf("Generate() error inesperado: %v", err)
	}

	if w := do(r, "/admin", "Bearer "+adminToken); w.Code != http.StatusNoContent {
		t.Fatalf("admin: status = %d, se esperaba 204", w.Code)
	}
	if w := do(r, "/admin", "Bearer "+customerToken); w.Code != http.StatusForbidden {
		t.Fatalf("customer: status = %d, se esperaba 403", w.Code)
	}
}
