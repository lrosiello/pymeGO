package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pymego/internal/domain"
)

func newOrderUC() (OrderUseCase, *fakeOrderRepo, *fakeProductRepo, *fakeTx) {
	return newOrderUCWithMode(domain.PriceModeB2C)
}

func newOrderUCWithMode(mode domain.PriceMode) (OrderUseCase, *fakeOrderRepo, *fakeProductRepo, *fakeTx) {
	orders := newFakeOrderRepo()
	products := newFakeProductRepo()
	tx := &fakeTx{}
	return NewOrderUseCase(orders, products, tx, "54 9 11 2233-4455", mode), orders, products, tx
}

func seedCatalog(products *fakeProductRepo, stock int) (*domain.Product, *domain.ProductVariant) {
	product := &domain.Product{ID: "prod-1", Name: "Remera", IsActive: true}
	products.products[product.ID] = product

	variant := &domain.ProductVariant{
		ID:            "var-1",
		ProductID:     product.ID,
		SKU:           "REM-ROJO-S",
		Name:          "Rojo / S",
		Price:         1500,
		StockQuantity: stock,
		IsActive:      true,
	}
	products.variants[variant.ID] = variant
	return product, variant
}

func TestOrderCreate_Success(t *testing.T) {
	uc, orders, products, tx := newOrderUC()
	_, variant := seedCatalog(products, 10)

	res, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "  Juan Pérez  ",
		CustomerPhone: "+54 9 11 2233-4455",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("Create() error inesperado: %v", err)
	}

	order := res.Order
	if order.ID == "" {
		t.Error("el pedido debería tener ID")
	}
	if order.CustomerName != "Juan Pérez" {
		t.Errorf("CustomerName = %q, se esperaba recortado", order.CustomerName)
	}
	if order.Status != domain.OrderStatusPending {
		t.Errorf("Status = %q, se esperaba pending", order.Status)
	}
	if order.TotalAmount != 3000 {
		t.Errorf("TotalAmount = %v, se esperaba 3000", order.TotalAmount)
	}
	if len(order.Items) != 1 {
		t.Fatalf("se esperaba 1 item, hay %d", len(order.Items))
	}
	item := order.Items[0]
	if item.ProductName != "Remera" || item.VariantName != "Rojo / S" {
		t.Errorf("snapshot inesperado: %+v", item)
	}
	if item.UnitPrice != 1500 {
		t.Errorf("UnitPrice = %v, se esperaba 1500", item.UnitPrice)
	}
	if products.variants[variant.ID].StockQuantity != 8 {
		t.Errorf("stock = %d, se esperaba 8", products.variants[variant.ID].StockQuantity)
	}
	if tx.calls != 1 {
		t.Errorf("se esperaba 1 transacción, hubo %d", tx.calls)
	}
	if !strings.HasPrefix(res.WhatsAppLink, "https://wa.me/5491122334455?text=") {
		t.Errorf("WhatsAppLink inesperado: %q", res.WhatsAppLink)
	}
	if !strings.Contains(res.WhatsAppText, "Total: $3000.00") {
		t.Errorf("WhatsAppText no contiene el total: %q", res.WhatsAppText)
	}
	if len(orders.orders) != 1 {
		t.Errorf("se esperaba 1 pedido persistido, hay %d", len(orders.orders))
	}
}

func TestOrderCreate_EmptyItems(t *testing.T) {
	uc, _, _, tx := newOrderUC()

	_, err := uc.Create(context.Background(), CreateOrderInput{CustomerName: "Juan", CustomerPhone: "123"})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Create() = %v, se esperaba ErrValidation", err)
	}
	if tx.calls != 0 {
		t.Errorf("no debería abrirse transacción para una entrada inválida (calls=%d)", tx.calls)
	}
}

func TestOrderCreate_MissingCustomer(t *testing.T) {
	uc, _, _, _ := newOrderUC()

	if _, err := uc.Create(context.Background(), CreateOrderInput{CustomerPhone: "123", Items: []CreateOrderItemInput{{VariantID: "var-1", Quantity: 1}}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Create() = %v, se esperaba ErrValidation por nombre faltante", err)
	}
	if _, err := uc.Create(context.Background(), CreateOrderInput{CustomerName: "Juan", Items: []CreateOrderItemInput{{VariantID: "var-1", Quantity: 1}}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Create() = %v, se esperaba ErrValidation por teléfono faltante", err)
	}
}

func TestOrderCreate_InvalidQuantity(t *testing.T) {
	uc, _, products, _ := newOrderUC()
	_, variant := seedCatalog(products, 10)

	_, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 0}},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Create() = %v, se esperaba ErrValidation", err)
	}
}

func TestOrderCreate_VariantNotFound(t *testing.T) {
	uc, _, _, _ := newOrderUC()

	_, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: "no-existe", Quantity: 1}},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Create() = %v, se esperaba ErrValidation", err)
	}
}

func TestOrderCreate_InactiveVariant(t *testing.T) {
	uc, _, products, _ := newOrderUC()
	_, variant := seedCatalog(products, 10)
	variant.IsActive = false

	_, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 1}},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Create() = %v, se esperaba ErrValidation", err)
	}
}

func TestOrderCreate_InsufficientStock(t *testing.T) {
	uc, _, products, _ := newOrderUC()
	_, variant := seedCatalog(products, 2)

	_, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 5}},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Create() = %v, se esperaba ErrValidation", err)
	}
	if products.variants[variant.ID].StockQuantity != 2 {
		t.Errorf("el stock no debe cambiar si la operación falla: %d", products.variants[variant.ID].StockQuantity)
	}
}

func TestOrderCreate_RepoError(t *testing.T) {
	uc, orders, products, _ := newOrderUC()
	_, variant := seedCatalog(products, 10)
	boom := errors.New("fallo de persistencia")
	orders.createErr = boom

	_, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 1}},
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Create() = %v, se esperaba el error del repositorio", err)
	}
}

func TestOrderUpdateStatus_CancelRestoresStock(t *testing.T) {
	uc, _, products, tx := newOrderUC()
	_, variant := seedCatalog(products, 10)

	res, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 3}},
	})
	if err != nil {
		t.Fatalf("Create() error inesperado: %v", err)
	}
	if got := products.variants[variant.ID].StockQuantity; got != 7 {
		t.Fatalf("stock tras crear = %d, se esperaba 7", got)
	}

	order, err := uc.UpdateStatus(context.Background(), res.Order.ID, domain.OrderStatusCancelled)
	if err != nil {
		t.Fatalf("UpdateStatus() error inesperado: %v", err)
	}
	if order.Status != domain.OrderStatusCancelled {
		t.Errorf("Status = %q, se esperaba cancelled", order.Status)
	}
	if got := products.variants[variant.ID].StockQuantity; got != 10 {
		t.Errorf("stock tras cancelar = %d, se esperaba 10", got)
	}
	if tx.calls != 2 {
		t.Errorf("se esperaban 2 transacciones (crear + cancelar), hubo %d", tx.calls)
	}
}

func TestOrderUpdateStatus_CancelIsIdempotent(t *testing.T) {
	uc, _, products, tx := newOrderUC()
	_, variant := seedCatalog(products, 10)

	res, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 3}},
	})
	if err != nil {
		t.Fatalf("Create() error inesperado: %v", err)
	}
	if _, err := uc.UpdateStatus(context.Background(), res.Order.ID, domain.OrderStatusCancelled); err != nil {
		t.Fatalf("primer UpdateStatus() error inesperado: %v", err)
	}

	callsAfterFirstCancel := tx.calls
	if _, err := uc.UpdateStatus(context.Background(), res.Order.ID, domain.OrderStatusCancelled); err != nil {
		t.Fatalf("segundo UpdateStatus() error inesperado: %v", err)
	}
	if got := products.variants[variant.ID].StockQuantity; got != 10 {
		t.Errorf("stock tras cancelar dos veces = %d, se esperaba 10", got)
	}
	if tx.calls != callsAfterFirstCancel {
		t.Errorf("una cancelación repetida no debería abrir transacción (calls=%d)", tx.calls)
	}
}

func TestOrderUpdateStatus_InvalidStatus(t *testing.T) {
	uc, _, _, _ := newOrderUC()

	_, err := uc.UpdateStatus(context.Background(), "order-1", domain.OrderStatus("inventado"))
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("UpdateStatus() = %v, se esperaba ErrValidation", err)
	}
}

func TestOrderUpdateStatus_NotFound(t *testing.T) {
	uc, _, _, _ := newOrderUC()

	_, err := uc.UpdateStatus(context.Background(), "no-existe", domain.OrderStatusConfirmed)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("UpdateStatus() = %v, se esperaba ErrNotFound", err)
	}
}

func TestOrderGet_BuildsLink(t *testing.T) {
	uc, orders, _, _ := newOrderUC()
	orders.orders["order-1"] = &domain.OrderLog{
		ID:            "order-1",
		CustomerName:  "Juan",
		CustomerPhone: "123",
		Status:        domain.OrderStatusPending,
		TotalAmount:   1500,
		Items: []domain.OrderItem{
			{VariantID: "var-1", ProductName: "Remera", VariantName: "Rojo / S", Quantity: 1, UnitPrice: 1500},
		},
	}

	res, err := uc.Get(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("Get() error inesperado: %v", err)
	}
	if !strings.HasPrefix(res.WhatsAppLink, "https://wa.me/5491122334455?text=") {
		t.Errorf("WhatsAppLink inesperado: %q", res.WhatsAppLink)
	}
	if !strings.Contains(res.WhatsAppText, "Remera") {
		t.Errorf("WhatsAppText no contiene el producto: %q", res.WhatsAppText)
	}
}

func b2bPrice(value float64) *float64 {
	return &value
}

func TestOrderCreate_B2BPricing(t *testing.T) {
	uc, _, products, _ := newOrderUCWithMode(domain.PriceModeB2B)
	_, variant := seedCatalog(products, 10)
	variant.B2BPrice = b2bPrice(1200)

	res, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Mayorista",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("Create() error inesperado: %v", err)
	}
	if res.Order.TotalAmount != 2400 {
		t.Errorf("TotalAmount = %v, se esperaba 2400", res.Order.TotalAmount)
	}
	if res.Order.Items[0].UnitPrice != 1200 {
		t.Errorf("UnitPrice = %v, se esperaba 1200", res.Order.Items[0].UnitPrice)
	}
	if !strings.Contains(res.WhatsAppText, "Total: $2400.00") {
		t.Errorf("WhatsAppText no contiene el total B2B: %q", res.WhatsAppText)
	}
}

func TestOrderCreate_B2BFallbackToRetail(t *testing.T) {
	uc, _, products, _ := newOrderUCWithMode(domain.PriceModeB2B)
	_, variant := seedCatalog(products, 10)
	variant.B2BPrice = nil

	res, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Mayorista",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("Create() error inesperado: %v", err)
	}
	if res.Order.TotalAmount != 1500 || res.Order.Items[0].UnitPrice != 1500 {
		t.Errorf("se esperaba fallback al precio minorista, total=%v unit=%v", res.Order.TotalAmount, res.Order.Items[0].UnitPrice)
	}
}

func TestOrderCreate_InactiveProduct(t *testing.T) {
	uc, _, products, _ := newOrderUC()
	product, variant := seedCatalog(products, 10)
	product.IsActive = false

	_, err := uc.Create(context.Background(), CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "123",
		Items:         []CreateOrderItemInput{{VariantID: variant.ID, Quantity: 1}},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Create() = %v, se esperaba ErrValidation por producto inactivo", err)
	}
}
