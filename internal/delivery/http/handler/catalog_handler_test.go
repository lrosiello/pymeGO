package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"pymego/internal/domain"
	"pymego/internal/usecase"
)

const testProductUUID = "11111111-1111-4111-8111-111111111111"

func sampleCatalogProduct() usecase.CatalogProduct {
	return usecase.CatalogProduct{
		Product: &domain.Product{
			ID:          testProductUUID,
			Name:        "Remera",
			Description: "Algodón",
			Images:      nil,
		},
		Variants: []usecase.CatalogVariant{
			{
				ID:            "22222222-2222-4222-8222-222222222222",
				Name:          "Rojo / S",
				Price:         1200,
				StockQuantity: 3,
				Available:     true,
				Attributes:    map[string]any{"color": "Rojo"},
			},
		},
	}
}

func TestCatalogHandler_ListCategories_OK(t *testing.T) {
	uc := &fakeCatalogUC{categoriesFn: func(context.Context) ([]domain.Category, error) {
		return []domain.Category{{ID: "cat-1", Name: "Ropa", Description: "Indumentaria", IsActive: true}}, nil
	}}
	h := NewCatalogHandler(uc, testLogger())

	w := performRequest(t, h.ListCategories, http.MethodGet, "/api/v1/catalog/categories", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200; body=%s", w.Code, w.Body.String())
	}

	var resp map[string][]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(resp["data"]) != 1 {
		t.Fatalf("data inesperada: %v", resp["data"])
	}
	if _, ok := resp["data"][0]["is_active"]; ok {
		t.Error("la respuesta pública no debe exponer is_active")
	}
}

func TestCatalogHandler_ListProducts_OK(t *testing.T) {
	uc := &fakeCatalogUC{productsFn: func(_ context.Context, categoryID *string) ([]usecase.CatalogProduct, error) {
		if categoryID != nil {
			t.Errorf("categoryID = %q, se esperaba nil", *categoryID)
		}
		return []usecase.CatalogProduct{sampleCatalogProduct()}, nil
	}}
	h := NewCatalogHandler(uc, testLogger())

	w := performRequest(t, h.ListProducts, http.MethodGet, "/api/v1/catalog/products", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200; body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data inesperada: %v", resp.Data)
	}

	variants, _ := resp.Data[0]["variants"].([]any)
	if len(variants) != 1 {
		t.Fatalf("variantes inesperadas: %v", resp.Data[0]["variants"])
	}
	variant, _ := variants[0].(map[string]any)
	if variant["price"] != float64(1200) {
		t.Errorf("price = %v, se esperaba 1200", variant["price"])
	}
	if _, ok := variant["b2b_price"]; ok {
		t.Error("la respuesta pública no debe exponer b2b_price")
	}
	if _, ok := variant["sku"]; ok {
		t.Error("la respuesta pública no debe exponer sku")
	}
}

func TestCatalogHandler_ListProducts_CategoryFilter(t *testing.T) {
	uc := &fakeCatalogUC{productsFn: func(_ context.Context, categoryID *string) ([]usecase.CatalogProduct, error) {
		if categoryID == nil || *categoryID != testProductUUID {
			t.Errorf("categoryID no propagado: %v", categoryID)
		}
		return []usecase.CatalogProduct{}, nil
	}}
	h := NewCatalogHandler(uc, testLogger())

	w := performRequest(t, h.ListProducts, http.MethodGet,
		"/api/v1/catalog/products?category_id="+testProductUUID, "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200; body=%s", w.Code, w.Body.String())
	}
}

func TestCatalogHandler_ListProducts_InvalidCategoryID(t *testing.T) {
	called := false
	uc := &fakeCatalogUC{productsFn: func(context.Context, *string) ([]usecase.CatalogProduct, error) {
		called = true
		return nil, nil
	}}
	h := NewCatalogHandler(uc, testLogger())

	w := performRequest(t, h.ListProducts, http.MethodGet,
		"/api/v1/catalog/products?category_id=no-uuid", "", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400; body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Error("no debería invocarse el caso de uso con un category_id inválido")
	}
}

func TestCatalogHandler_GetProduct_OK(t *testing.T) {
	uc := &fakeCatalogUC{getFn: func(_ context.Context, id string) (*usecase.CatalogProduct, error) {
		if id != testProductUUID {
			t.Errorf("id = %q, se esperaba %q", id, testProductUUID)
		}
		product := sampleCatalogProduct()
		return &product, nil
	}}
	h := NewCatalogHandler(uc, testLogger())

	w := performRequest(t, h.GetProduct, http.MethodGet, "/api/v1/catalog/products/"+testProductUUID, "",
		gin.Params{{Key: "id", Value: testProductUUID}})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200; body=%s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if body["name"] != "Remera" {
		t.Errorf("name = %v", body["name"])
	}
	if _, ok := body["is_active"]; ok {
		t.Error("la respuesta pública no debe exponer is_active")
	}
	images, _ := body["images"].([]any)
	if images == nil {
		t.Error("images debería serializarse como array vacío, no null")
	}
}

func TestCatalogHandler_GetProduct_InvalidID(t *testing.T) {
	h := NewCatalogHandler(&fakeCatalogUC{}, testLogger())

	w := performRequest(t, h.GetProduct, http.MethodGet, "/api/v1/catalog/products/no-uuid", "",
		gin.Params{{Key: "id", Value: "no-uuid"}})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400; body=%s", w.Code, w.Body.String())
	}
}

func TestCatalogHandler_GetProduct_NotFound(t *testing.T) {
	uc := &fakeCatalogUC{getFn: func(context.Context, string) (*usecase.CatalogProduct, error) {
		return nil, domain.ErrNotFound
	}}
	h := NewCatalogHandler(uc, testLogger())

	w := performRequest(t, h.GetProduct, http.MethodGet, "/api/v1/catalog/products/"+testProductUUID, "",
		gin.Params{{Key: "id", Value: testProductUUID}})
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, se esperaba 404; body=%s", w.Code, w.Body.String())
	}
}
