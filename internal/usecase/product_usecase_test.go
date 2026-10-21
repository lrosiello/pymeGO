package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pymego/internal/domain"
)

func newProductUC() (ProductUseCase, *fakeCategoryRepo, *fakeProductRepo, *fakeTx) {
	categories := newFakeCategoryRepo()
	products := newFakeProductRepo()
	tx := &fakeTx{}
	return NewProductUseCase(categories, products, tx), categories, products, tx
}

func TestCreateCategory_Success(t *testing.T) {
	uc, _, _, _ := newProductUC()

	c, err := uc.CreateCategory(context.Background(), CreateCategoryInput{
		Name:        "  Ropa  ",
		Description: "  Indumentaria  ",
	})
	if err != nil {
		t.Fatalf("CreateCategory() error inesperado: %v", err)
	}
	if c.ID == "" {
		t.Error("la categoría debería tener ID")
	}
	if c.Name != "Ropa" {
		t.Errorf("Name = %q, se esperaba Ropa (sin espacios)", c.Name)
	}
	if c.Description != "Indumentaria" {
		t.Errorf("Description = %q, se esperaba sin espacios", c.Description)
	}
	if !c.IsActive {
		t.Error("la categoría debería crearse activa")
	}
}

func TestCreateCategory_EmptyName(t *testing.T) {
	uc, _, _, _ := newProductUC()

	_, err := uc.CreateCategory(context.Background(), CreateCategoryInput{Name: "   "})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("CreateCategory() = %v, se esperaba ErrValidation", err)
	}
}

func TestCreateCategory_ParentNotFound(t *testing.T) {
	uc, _, _, _ := newProductUC()
	parent := "no-existe"

	_, err := uc.CreateCategory(context.Background(), CreateCategoryInput{Name: "Ropa", ParentID: &parent})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("CreateCategory() = %v, se esperaba ErrValidation", err)
	}
}

func TestUpdateCategory_SelfParent(t *testing.T) {
	uc, categories, _, _ := newProductUC()
	parent := &domain.Category{ID: "cat-1", Name: "Ropa", IsActive: true}
	categories.items[parent.ID] = parent

	_, err := uc.UpdateCategory(context.Background(), parent.ID, UpdateCategoryInput{Name: "Ropa", ParentID: &parent.ID, IsActive: true})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("UpdateCategory() = %v, se esperaba ErrValidation", err)
	}
}

func TestCreateProduct_WithVariants(t *testing.T) {
	uc, categories, _, tx := newProductUC()
	category := &domain.Category{ID: "cat-1", Name: "Ropa", IsActive: true}
	categories.items[category.ID] = category

	detail, err := uc.CreateProduct(context.Background(), CreateProductInput{
		CategoryID: &category.ID,
		Name:       "  Remera  ",
		IsActive:   true,
		Variants: []CreateVariantInput{
			{SKU: "REM-ROJO-S", Name: "Rojo / S", Price: 1500, StockQuantity: 5},
			{SKU: "REM-ROJO-M", Name: "Rojo / M", Price: 1500, StockQuantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct() error inesperado: %v", err)
	}

	if tx.calls != 1 {
		t.Errorf("se esperaba 1 transacción, hubo %d", tx.calls)
	}
	if detail.Product.ID == "" {
		t.Error("el producto debería tener ID")
	}
	if detail.Product.Name != "Remera" {
		t.Errorf("Name = %q, se esperaba Remera", detail.Product.Name)
	}
	if len(detail.Variants) != 2 {
		t.Fatalf("se esperaban 2 variantes, hay %d", len(detail.Variants))
	}
	for _, v := range detail.Variants {
		if v.ProductID != detail.Product.ID {
			t.Errorf("variante %q asociada a %q, se esperaba %q", v.ID, v.ProductID, detail.Product.ID)
		}
		if !v.IsActive {
			t.Errorf("la variante %q debería crearse activa", v.SKU)
		}
		if v.Attributes == nil {
			t.Errorf("la variante %q debería tener Attributes no nulo", v.SKU)
		}
	}
}

func TestCreateProduct_InvalidVariant(t *testing.T) {
	uc, _, _, _ := newProductUC()

	_, err := uc.CreateProduct(context.Background(), CreateProductInput{
		Name:     "Remera",
		IsActive: true,
		Variants: []CreateVariantInput{
			{SKU: "", Name: "Rojo / S", Price: 1500, StockQuantity: 5},
		},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("CreateProduct() = %v, se esperaba ErrValidation", err)
	}
	if !strings.Contains(err.Error(), "variante[0]") {
		t.Errorf("el error debería identificar la variante inválida, fue: %v", err)
	}
}

func TestCreateProduct_CategoryNotFound(t *testing.T) {
	uc, _, _, _ := newProductUC()
	categoryID := "no-existe"

	_, err := uc.CreateProduct(context.Background(), CreateProductInput{Name: "Remera", CategoryID: &categoryID})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("CreateProduct() = %v, se esperaba ErrValidation", err)
	}
}

func TestAddVariant_ProductNotFound(t *testing.T) {
	uc, _, _, _ := newProductUC()

	_, err := uc.AddVariant(context.Background(), "no-existe", CreateVariantInput{SKU: "S1", Name: "N", Price: 10})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("AddVariant() = %v, se esperaba ErrNotFound", err)
	}
}

func TestAddVariant_DuplicateSKU(t *testing.T) {
	uc, _, products, _ := newProductUC()
	product := &domain.Product{ID: "prod-1", Name: "Remera", IsActive: true}
	products.products[product.ID] = product
	products.variants["var-1"] = &domain.ProductVariant{ID: "var-1", ProductID: product.ID, SKU: "SKU-1", Name: "N", Price: 10}

	_, err := uc.AddVariant(context.Background(), product.ID, CreateVariantInput{SKU: "SKU-1", Name: "Otra", Price: 20})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("AddVariant() = %v, se esperaba ErrConflict", err)
	}
}

func TestListVariants_ProductNotFound(t *testing.T) {
	uc, _, _, _ := newProductUC()

	if _, err := uc.ListVariants(context.Background(), "no-existe"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ListVariants() = %v, se esperaba ErrNotFound", err)
	}
}

func TestAdjustStock(t *testing.T) {
	uc, _, products, _ := newProductUC()
	products.variants["var-1"] = &domain.ProductVariant{ID: "var-1", SKU: "S1", Name: "N", Price: 10, StockQuantity: 10}

	v, err := uc.AdjustStock(context.Background(), "var-1", -3)
	if err != nil {
		t.Fatalf("AdjustStock() error inesperado: %v", err)
	}
	if v.StockQuantity != 7 {
		t.Errorf("stock = %d, se esperaba 7", v.StockQuantity)
	}
	if products.variants["var-1"].StockQuantity != 7 {
		t.Errorf("el stock persistido no se actualizó: %d", products.variants["var-1"].StockQuantity)
	}

	if _, err := uc.AdjustStock(context.Background(), "var-1", -20); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("AdjustStock() = %v, se esperaba ErrValidation por stock insuficiente", err)
	}
	if products.variants["var-1"].StockQuantity != 7 {
		t.Errorf("el stock no debe cambiar tras un ajuste inválido: %d", products.variants["var-1"].StockQuantity)
	}

	if _, err := uc.AdjustStock(context.Background(), "var-1", 0); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("AdjustStock(0) = %v, se esperaba ErrValidation", err)
	}
}

func TestUpdateVariant_NotFound(t *testing.T) {
	uc, _, _, _ := newProductUC()

	_, err := uc.UpdateVariant(context.Background(), "no-existe", UpdateVariantInput{SKU: "S1", Name: "N", Price: 10})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("UpdateVariant() = %v, se esperaba ErrNotFound", err)
	}
}

func TestUpdateVariant_EmptySKU(t *testing.T) {
	uc, _, products, _ := newProductUC()
	products.variants["var-1"] = &domain.ProductVariant{ID: "var-1", SKU: "S1", Name: "N", Price: 10}

	_, err := uc.UpdateVariant(context.Background(), "var-1", UpdateVariantInput{SKU: " ", Name: "N", Price: 10})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("UpdateVariant() = %v, se esperaba ErrValidation", err)
	}
}
