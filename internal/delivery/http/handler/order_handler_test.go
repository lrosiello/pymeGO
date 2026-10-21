package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pymego/internal/domain"
	"pymego/internal/usecase"
)

const testVariantUUID = "550e8400-e29b-41d4-a716-446655440000"

func sampleOrder() *domain.OrderLog {
	return &domain.OrderLog{
		ID:            "order-1",
		CustomerName:  "Juan",
		CustomerPhone: "123",
		TotalAmount:   1500,
		Status:        domain.OrderStatusPending,
		Items: []domain.OrderItem{
			{VariantID: testVariantUUID, ProductName: "Remera", VariantName: "Rojo / S", Quantity: 1, UnitPrice: 1500},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestOrderHandler_Create_Created(t *testing.T) {
	uc := &fakeOrderUC{createFn: func(_ context.Context, in usecase.CreateOrderInput) (*usecase.OrderResult, error) {
		if len(in.Items) != 1 || in.Items[0].VariantID != testVariantUUID || in.Items[0].Quantity != 2 {
			t.Errorf("input no propagado correctamente: %+v", in)
		}
		return &usecase.OrderResult{
			Order:        sampleOrder(),
			WhatsAppText: "Pedido de Juan",
			WhatsAppLink: "https://wa.me/5491122334455?text=pedido",
		}, nil
	}}
	h := NewOrderHandler(uc, testLogger())

	body := `{"customer_name":"Juan","customer_phone":"123","items":[{"variant_id":"` + testVariantUUID + `","quantity":2}]}`
	w := performRequest(t, h.Create, http.MethodPost, "/api/v1/orders", body, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201; body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if resp["whatsapp_link"] != "https://wa.me/5491122334455?text=pedido" {
		t.Errorf("whatsapp_link inesperado: %v", resp["whatsapp_link"])
	}
	if resp["status"] != "pending" {
		t.Errorf("status = %v, se esperaba pending", resp["status"])
	}
}

func TestOrderHandler_Create_InvalidBody(t *testing.T) {
	h := NewOrderHandler(&fakeOrderUC{}, testLogger())

	w := performRequest(t, h.Create, http.MethodPost, "/api/v1/orders",
		`{"customer_name":"Juan","customer_phone":"123","items":[]}`, nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400; body=%s", w.Code, w.Body.String())
	}
}

func TestOrderHandler_Create_InvalidUUID(t *testing.T) {
	h := NewOrderHandler(&fakeOrderUC{}, testLogger())

	w := performRequest(t, h.Create, http.MethodPost, "/api/v1/orders",
		`{"customer_name":"Juan","customer_phone":"123","items":[{"variant_id":"no-uuid","quantity":1}]}`, nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400; body=%s", w.Code, w.Body.String())
	}
}

func TestOrderHandler_Get_NotFound(t *testing.T) {
	uc := &fakeOrderUC{getFn: func(context.Context, string) (*usecase.OrderResult, error) {
		return nil, domain.ErrNotFound
	}}
	h := NewOrderHandler(uc, testLogger())

	w := performRequest(t, h.Get, http.MethodGet, "/api/v1/orders/order-1", "",
		gin.Params{{Key: "id", Value: "order-1"}})

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, se esperaba 404; body=%s", w.Code, w.Body.String())
	}
}

func TestOrderHandler_Get_MissingID(t *testing.T) {
	h := NewOrderHandler(&fakeOrderUC{}, testLogger())

	w := performRequest(t, h.Get, http.MethodGet, "/api/v1/orders/", "", nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400; body=%s", w.Code, w.Body.String())
	}
}

func TestOrderHandler_UpdateStatus_OK(t *testing.T) {
	uc := &fakeOrderUC{updateStatusFn: func(_ context.Context, id string, status domain.OrderStatus) (*domain.OrderLog, error) {
		if id != "order-1" || status != domain.OrderStatusCancelled {
			t.Errorf("argumentos inesperados: id=%q status=%q", id, status)
		}
		order := sampleOrder()
		order.Status = status
		return order, nil
	}}
	h := NewOrderHandler(uc, testLogger())

	w := performRequest(t, h.UpdateStatus, http.MethodPatch, "/api/v1/orders/order-1/status",
		`{"status":"cancelled"}`, gin.Params{{Key: "id", Value: "order-1"}})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200; body=%s", w.Code, w.Body.String())
	}
}

func TestOrderHandler_UpdateStatus_InvalidStatus(t *testing.T) {
	h := NewOrderHandler(&fakeOrderUC{}, testLogger())

	w := performRequest(t, h.UpdateStatus, http.MethodPatch, "/api/v1/orders/order-1/status",
		`{"status":"inventado"}`, gin.Params{{Key: "id", Value: "order-1"}})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400; body=%s", w.Code, w.Body.String())
	}
}

func TestOrderHandler_List_OK(t *testing.T) {
	uc := &fakeOrderUC{listFn: func(context.Context) ([]domain.OrderLog, error) {
		return []domain.OrderLog{*sampleOrder()}, nil
	}}
	h := NewOrderHandler(uc, testLogger())

	w := performRequest(t, h.List, http.MethodGet, "/api/v1/orders", "", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200; body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	data, ok := resp["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data inesperada: %v", resp["data"])
	}
}
