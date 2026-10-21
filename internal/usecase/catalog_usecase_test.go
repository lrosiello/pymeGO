package usecase

import (
	"context"
	"errors"
	"testing"

	"pymego/internal/domain"
)

func stringPtr(value string) *string {
	return &value
}

func seedCatalogProduct(
	products *fakeProductRepo,
	id, name string,
	active bool,
	categoryID *string,
) *domain.Product {
	product := &domain.Product{ID: id, Name: name, IsActive: active, CategoryID: categoryID}
	products.products[id] = product
	return product
}

func seedCatalogVariant(
	products *fakeProductRepo,
	id, productID string,
	active bool,
	stock int,
	price float64,
	b2bPrice *float64,
) *domain.ProductVariant {
	variant := &domain.ProductVariant{
		ID:            id,
		ProductID:     productID,
		SKU:           "SKU-" + id,
		Name:          "Variante " + id,
		Price:         price,
		B2BPrice:      b2bPrice,
		StockQuantity: stock,
		IsActive:      active,
		Attributes:    map[string]any{"color": "Rojo"},
	}
	products.variants[id] = variant
	return variant
}

func newCatalogUC(mode domain.PriceMode) (CatalogUseCase, *fakeCategoryRepo, *fakeProductRepo) {
	categories := newFakeCategoryRepo()
	products := newFakeProductRepo()
	return NewCatalogUseCase(categories, products, mode), categories, products
}

func TestCatalogListCategories_OnlyActive(t *testing.T) {
	uc, categories, _ := newCatalogUC(domain.PriceModeB2C)
	categories.items["cat-1"] = &domain.Category{ID: "cat-1", Name: "Activa", IsActive: true}
	categories.items["cat-2"] = &domain.Category{ID: "cat-2", Name: "Inactiva", IsActive: false}

	got, err := uc.ListCategories(context.Background())
	if err != nil {
		t.Fatalf("ListCategories() error inesperado: %v", err)
	}
	if len(got) != 1 || got[0].ID != "cat-1" {
		t.Fatalf("ListCategories() = %+v, se esperaba solo la activa", got)
	}
}

func TestCatalogListProducts_FiltersNonSellable(t *testing.T) {
	uc, _, products := newCatalogUC(domain.PriceModeB2C)

	seedCatalogProduct(products, "vis", "Visible", true, nil)
	seedCatalogVariant(products, "v-vis", "vis", true, 5, 1500, nil)

	seedCatalogProduct(products, "inactive", "Inactivo", false, nil)
	seedCatalogVariant(products, "v-inactive", "inactive", true, 5, 1000, nil)

	seedCatalogProduct(products, "nostock", "Sin stock", true, nil)
	seedCatalogVariant(products, "v-nostock", "nostock", true, 0, 1000, nil)

	seedCatalogProduct(products, "inactivevar", "Variante inactiva", true, nil)
	seedCatalogVariant(products, "v-inactivevar", "inactivevar", false, 5, 1000, nil)

	got, err := uc.ListProducts(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListProducts() error inesperado: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListProducts() devolvió %d productos, se esperaba 1", len(got))
	}
	if got[0].Product.ID != "vis" {
		t.Errorf("producto inesperado: %q", got[0].Product.ID)
	}
	if len(got[0].Variants) != 1 || !got[0].Variants[0].Available {
		t.Errorf("variantes inesperadas: %+v", got[0].Variants)
	}
}

func TestCatalogListProducts_B2CPrice(t *testing.T) {
	uc, _, products := newCatalogUC(domain.PriceModeB2C)
	seedCatalogProduct(products, "p1", "Producto", true, nil)
	seedCatalogVariant(products, "v1", "p1", true, 5, 1500, b2bPrice(1200))

	got, err := uc.ListProducts(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListProducts() error inesperado: %v", err)
	}
	if got[0].Variants[0].Price != 1500 {
		t.Errorf("Price = %v, se esperaba el minorista 1500", got[0].Variants[0].Price)
	}
}

func TestCatalogListProducts_B2BPrice(t *testing.T) {
	uc, _, products := newCatalogUC(domain.PriceModeB2B)
	seedCatalogProduct(products, "p1", "Producto", true, nil)
	seedCatalogVariant(products, "v1", "p1", true, 5, 1500, b2bPrice(1200))

	got, err := uc.ListProducts(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListProducts() error inesperado: %v", err)
	}
	if got[0].Variants[0].Price != 1200 {
		t.Errorf("Price = %v, se esperaba el mayorista 1200", got[0].Variants[0].Price)
	}
}

func TestCatalogListProducts_B2BFallback(t *testing.T) {
	uc, _, products := newCatalogUC(domain.PriceModeB2B)
	seedCatalogProduct(products, "p1", "Producto", true, nil)
	seedCatalogVariant(products, "v1", "p1", true, 5, 1500, nil)

	got, err := uc.ListProducts(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListProducts() error inesperado: %v", err)
	}
	if got[0].Variants[0].Price != 1500 {
		t.Errorf("Price = %v, se esperaba fallback al minorista 1500", got[0].Variants[0].Price)
	}
}

func TestCatalogListProducts_CategoryFilter(t *testing.T) {
	uc, _, products := newCatalogUC(domain.PriceModeB2C)
	catA := stringPtr("cat-a")
	catB := stringPtr("cat-b")

	seedCatalogProduct(products, "pa", "Producto A", true, catA)
	seedCatalogVariant(products, "va", "pa", true, 5, 1000, nil)
	seedCatalogProduct(products, "pb", "Producto B", true, catB)
	seedCatalogVariant(products, "vb", "pb", true, 5, 1000, nil)

	got, err := uc.ListProducts(context.Background(), catA)
	if err != nil {
		t.Fatalf("ListProducts() error inesperado: %v", err)
	}
	if len(got) != 1 || got[0].Product.ID != "pa" {
		t.Fatalf("ListProducts(cat-a) = %+v, se esperaba solo pa", got)
	}
}

func TestCatalogGetProduct_NotFound(t *testing.T) {
	uc, _, products := newCatalogUC(domain.PriceModeB2C)

	seedCatalogProduct(products, "inactive", "Inactivo", false, nil)
	seedCatalogVariant(products, "v-inactive", "inactive", true, 5, 1000, nil)

	seedCatalogProduct(products, "nostock", "Sin stock", true, nil)
	seedCatalogVariant(products, "v-nostock", "nostock", true, 0, 1000, nil)

	cases := []string{"no-existe", "inactive", "nostock"}
	for _, id := range cases {
		if _, err := uc.GetProduct(context.Background(), id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetProduct(%q) = %v, se esperaba ErrNotFound", id, err)
		}
	}
}

func TestCatalogGetProduct_ReturnsActiveVariants(t *testing.T) {
	uc, _, products := newCatalogUC(domain.PriceModeB2C)
	seedCatalogProduct(products, "p1", "Producto", true, nil)
	seedCatalogVariant(products, "v1", "p1", true, 5, 1500, nil)
	seedCatalogVariant(products, "v2", "p1", true, 0, 1600, nil)
	seedCatalogVariant(products, "v3", "p1", false, 5, 1700, nil)

	got, err := uc.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("GetProduct() error inesperado: %v", err)
	}
	if len(got.Variants) != 2 {
		t.Fatalf("se esperaban 2 variantes activas, hay %d", len(got.Variants))
	}

	byID := map[string]CatalogVariant{}
	for _, v := range got.Variants {
		byID[v.ID] = v
	}
	if !byID["v1"].Available {
		t.Error("v1 debería estar disponible")
	}
	if byID["v2"].Available {
		t.Error("v2 no debería estar disponible (stock 0)")
	}
	if _, ok := byID["v3"]; ok {
		t.Error("v3 está inactiva y no debería exponerse")
	}
}
