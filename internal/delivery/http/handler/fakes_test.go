package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"pymego/internal/domain"
	"pymego/internal/usecase"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeAuthUC struct {
	registerFn func(ctx context.Context, in usecase.RegisterInput) (*usecase.AuthResult, error)
	loginFn    func(ctx context.Context, in usecase.LoginInput) (*usecase.AuthResult, error)
}

func (f *fakeAuthUC) Register(ctx context.Context, in usecase.RegisterInput) (*usecase.AuthResult, error) {
	if f.registerFn != nil {
		return f.registerFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeAuthUC) Login(ctx context.Context, in usecase.LoginInput) (*usecase.AuthResult, error) {
	if f.loginFn != nil {
		return f.loginFn(ctx, in)
	}
	return nil, nil
}

type fakeOrderUC struct {
	createFn       func(ctx context.Context, in usecase.CreateOrderInput) (*usecase.OrderResult, error)
	listFn         func(ctx context.Context) ([]domain.OrderLog, error)
	getFn          func(ctx context.Context, id string) (*usecase.OrderResult, error)
	updateStatusFn func(ctx context.Context, id string, status domain.OrderStatus) (*domain.OrderLog, error)
}

func (f *fakeOrderUC) Create(ctx context.Context, in usecase.CreateOrderInput) (*usecase.OrderResult, error) {
	if f.createFn != nil {
		return f.createFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeOrderUC) List(ctx context.Context) ([]domain.OrderLog, error) {
	if f.listFn != nil {
		return f.listFn(ctx)
	}
	return nil, nil
}

func (f *fakeOrderUC) Get(ctx context.Context, id string) (*usecase.OrderResult, error) {
	if f.getFn != nil {
		return f.getFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeOrderUC) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) (*domain.OrderLog, error) {
	if f.updateStatusFn != nil {
		return f.updateStatusFn(ctx, id, status)
	}
	return nil, nil
}

// performRequest ejecuta un handler de Gin de forma aislada (sin router),
// inyectando body JSON y parámetros de ruta.
func performRequest(t *testing.T, handler gin.HandlerFunc, method, target, body string, params gin.Params) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = params

	handler(c)
	return w
}
