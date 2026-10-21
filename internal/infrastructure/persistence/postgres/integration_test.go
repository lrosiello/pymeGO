//go:build integration

// Package postgres contiene tests de integración que se ejecutan contra una base
// de datos PostgreSQL real. Se activan con la etiqueta de compilación
// "integration" y requieren la variable de entorno TEST_DATABASE_URL:
//
//	go test -tags=integration ./internal/infrastructure/persistence/postgres/...
//
// Si TEST_DATABASE_URL no está definida, el paquete termina sin ejecutar tests.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"pymego/internal/domain"
	"pymego/internal/usecase"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL no definida: se omiten los tests de integración")
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo crear el pool de pruebas: %v\n", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo conectar a TEST_DATABASE_URL: %v\n", err)
		pool.Close()
		os.Exit(1)
	}
	testPool = pool

	if err := resetSchema(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo preparar el schema: %v\n", err)
		pool.Close()
		os.Exit(1)
	}

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// resetSchema elimina las tablas y aplica la migración inicial.
func resetSchema(ctx context.Context) error {
	drop := `DROP TABLE IF EXISTS order_logs, product_variants, products, categories, users CASCADE`
	if _, err := testPool.Exec(ctx, drop); err != nil {
		return fmt.Errorf("drop de tablas: %w", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("no se pudo localizar el archivo de test")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "migrations", "0001_init.sql")

	script, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("lectura de la migración %s: %w", path, err)
	}

	return execScript(ctx, string(script))
}

// execScript ejecuta un script SQL separándolo por sentencias. La migración no
// contiene punto y coma dentro de literales, por lo que el corte es seguro.
func execScript(ctx context.Context, script string) error {
	for _, stmt := range strings.Split(script, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := testPool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("ejecutando sentencia: %w", err)
		}
	}
	return nil
}

// truncateTables deja las tablas vacías antes de cada test.
func truncateTables(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(),
		`TRUNCATE order_logs, product_variants, products, categories, users CASCADE`); err != nil {
		t.Fatalf("no se pudieron truncar las tablas: %v", err)
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestIntegration_UserRepository(t *testing.T) {
	truncateTables(t)
	ctx := testContext(t)
	repo := NewUserRepository(testPool)

	user := &domain.User{
		Email:        "owner@integration.test",
		PasswordHash: "hash-de-prueba",
		FullName:     "Owner",
		Role:         domain.RoleAdmin,
		IsActive:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error inesperado: %v", err)
	}
	if user.ID == "" {
		t.Fatal("Create() no rellenó el ID")
	}
	if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
		t.Fatal("Create() no rellenó los timestamps")
	}

	byEmail, err := repo.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("GetByEmail() error inesperado: %v", err)
	}
	if byEmail.ID != user.ID || byEmail.Role != domain.RoleAdmin || !byEmail.IsActive {
		t.Errorf("GetByEmail() inesperado: %+v", byEmail)
	}

	byID, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error inesperado: %v", err)
	}
	if byID.Email != user.Email {
		t.Errorf("GetByID().Email = %q, se esperaba %q", byID.Email, user.Email)
	}

	exists, err := repo.ExistsByEmail(ctx, user.Email)
	if err != nil || !exists {
		t.Errorf("ExistsByEmail() = %v, %v; se esperaba true, nil", exists, err)
	}
	exists, err = repo.ExistsByEmail(ctx, "nadie@integration.test")
	if err != nil || exists {
		t.Errorf("ExistsByEmail() = %v, %v; se esperaba false, nil", exists, err)
	}

	duplicate := &domain.User{
		Email: user.Email, PasswordHash: "otro", FullName: "Duplicado", Role: domain.RoleStaff, IsActive: true,
	}
	if err := repo.Create(ctx, duplicate); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Create() duplicado = %v, se esperaba ErrConflict", err)
	}

	if _, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByID() inexistente = %v, se esperaba ErrNotFound", err)
	}
}

func TestIntegration_CategoryRepository(t *testing.T) {
	truncateTables(t)
	ctx := testContext(t)
	repo := NewCategoryRepository(testPool)

	root := &domain.Category{Name: "Ropa", Description: "Indumentaria", IsActive: true}
	if err := repo.Create(ctx, root); err != nil {
		t.Fatalf("Create() raíz error inesperado: %v", err)
	}

	child := &domain.Category{ParentID: &root.ID, Name: "Remeras", IsActive: true}
	if err := repo.Create(ctx, child); err != nil {
		t.Fatalf("Create() hija error inesperado: %v", err)
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error inesperado: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List() devolvió %d categorías, se esperaban 2", len(list))
	}

	got, err := repo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetByID() error inesperado: %v", err)
	}
	if got.ParentID == nil || *got.ParentID != root.ID {
		t.Errorf("ParentID inesperado: %v", got.ParentID)
	}

	got.Name = "Remeras manga corta"
	got.IsActive = false
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update() error inesperado: %v", err)
	}
	updated, err := repo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetByID() tras update error inesperado: %v", err)
	}
	if updated.Name != "Remeras manga corta" || updated.IsActive {
		t.Errorf("Update() no persistió los cambios: %+v", updated)
	}

	if err := repo.Delete(ctx, child.ID); err != nil {
		t.Fatalf("Delete() error inesperado: %v", err)
	}
	if _, err := repo.GetByID(ctx, child.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByID() tras delete = %v, se esperaba ErrNotFound", err)
	}
	if err := repo.Delete(ctx, child.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Delete() repetido = %v, se esperaba ErrNotFound", err)
	}
}

func TestIntegration_ProductRepository(t *testing.T) {
	truncateTables(t)
	ctx := testContext(t)
	repo := NewProductRepository(testPool)

	product := &domain.Product{
		Name:        "Remera básica",
		Description: "Algodón",
		Images:      []string{"https://example.com/remera.jpg"},
		IsActive:    true,
	}
	if err := repo.Create(ctx, product); err != nil {
		t.Fatalf("Create() producto error inesperado: %v", err)
	}
	if product.ID == "" {
		t.Fatal("Create() producto no rellenó el ID")
	}

	variant := &domain.ProductVariant{
		ProductID:     product.ID,
		SKU:           "REM-ROJO-S",
		Name:          "Rojo / S",
		Price:         1500,
		StockQuantity: 10,
		Attributes:    map[string]any{"color": "Rojo", "talle": "S"},
		IsActive:      true,
	}
	if err := repo.CreateVariant(ctx, variant); err != nil {
		t.Fatalf("CreateVariant() error inesperado: %v", err)
	}

	gotProduct, err := repo.GetByID(ctx, product.ID)
	if err != nil {
		t.Fatalf("GetByID() producto error inesperado: %v", err)
	}
	if gotProduct.Name != product.Name || !reflect.DeepEqual(gotProduct.Images, product.Images) {
		t.Errorf("GetByID() producto inesperado: %+v", gotProduct)
	}

	products, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error inesperado: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("List() devolvió %d productos, se esperaba 1", len(products))
	}

	variants, err := repo.ListVariantsByProduct(ctx, product.ID)
	if err != nil {
		t.Fatalf("ListVariantsByProduct() error inesperado: %v", err)
	}
	if len(variants) != 1 {
		t.Fatalf("ListVariantsByProduct() devolvió %d variantes, se esperaba 1", len(variants))
	}
	if !reflect.DeepEqual(variants[0].Attributes, variant.Attributes) {
		t.Errorf("Attributes inesperados: %v", variants[0].Attributes)
	}

	duplicate := &domain.ProductVariant{
		ProductID: product.ID, SKU: variant.SKU, Name: "Duplicada", Price: 1, IsActive: true,
	}
	if err := repo.CreateVariant(ctx, duplicate); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("CreateVariant() SKU duplicado = %v, se esperaba ErrConflict", err)
	}

	gotVariant, err := repo.GetVariantByID(ctx, variant.ID)
	if err != nil {
		t.Fatalf("GetVariantByID() error inesperado: %v", err)
	}
	gotVariant.Name = "Rojo / Small"
	gotVariant.Price = 1600
	if err := repo.UpdateVariant(ctx, gotVariant); err != nil {
		t.Fatalf("UpdateVariant() error inesperado: %v", err)
	}
	updated, err := repo.GetVariantByID(ctx, variant.ID)
	if err != nil {
		t.Fatalf("GetVariantByID() tras update error inesperado: %v", err)
	}
	if updated.Name != "Rojo / Small" || updated.Price != 1600 {
		t.Errorf("UpdateVariant() no persistió los cambios: %+v", updated)
	}
	if updated.StockQuantity != 10 {
		t.Errorf("UpdateVariant() no debe tocar el stock: %d", updated.StockQuantity)
	}

	adjusted, err := repo.AdjustStockQuantity(ctx, variant.ID, -3)
	if err != nil {
		t.Fatalf("AdjustStockQuantity(-3) error inesperado: %v", err)
	}
	if adjusted.StockQuantity != 7 {
		t.Errorf("stock tras -3 = %d, se esperaba 7", adjusted.StockQuantity)
	}

	if _, err := repo.AdjustStockQuantity(ctx, variant.ID, -100); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("AdjustStockQuantity(-100) = %v, se esperaba ErrValidation", err)
	}
	after, err := repo.GetVariantByID(ctx, variant.ID)
	if err != nil {
		t.Fatalf("GetVariantByID() error inesperado: %v", err)
	}
	if after.StockQuantity != 7 {
		t.Errorf("el stock no debe cambiar tras un ajuste inválido: %d", after.StockQuantity)
	}

	if _, err := repo.AdjustStockQuantity(ctx, "00000000-0000-0000-0000-000000000000", -1); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("AdjustStockQuantity() inexistente = %v, se esperaba ErrNotFound", err)
	}

	if err := repo.Delete(ctx, product.ID); err != nil {
		t.Fatalf("Delete() producto error inesperado: %v", err)
	}
	if _, err := repo.GetByID(ctx, product.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByID() tras delete = %v, se esperaba ErrNotFound", err)
	}
	remaining, err := repo.ListVariantsByProduct(ctx, product.ID)
	if err != nil {
		t.Fatalf("ListVariantsByProduct() tras delete error inesperado: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("las variantes deberían eliminarse en cascada, quedan %d", len(remaining))
	}
}

func TestIntegration_OrderRepository(t *testing.T) {
	truncateTables(t)
	ctx := testContext(t)
	repo := NewOrderRepository(testPool)

	order := &domain.OrderLog{
		CustomerName:  "Juan Pérez",
		CustomerPhone: "+5491122334455",
		TotalAmount:   3000,
		Status:        domain.OrderStatusPending,
		Items: []domain.OrderItem{
			{VariantID: "v1", ProductName: "Remera", VariantName: "Rojo / S", Quantity: 2, UnitPrice: 1500},
		},
	}
	if err := repo.Create(ctx, order); err != nil {
		t.Fatalf("Create() error inesperado: %v", err)
	}
	if order.ID == "" || order.CreatedAt.IsZero() {
		t.Fatal("Create() no rellenó ID/timestamps")
	}

	got, err := repo.GetByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetByID() error inesperado: %v", err)
	}
	if got.CustomerName != order.CustomerName || got.TotalAmount != order.TotalAmount || got.Status != domain.OrderStatusPending {
		t.Errorf("GetByID() inesperado: %+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].ProductName != "Remera" || got.Items[0].UnitPrice != 1500 {
		t.Errorf("items inesperados: %+v", got.Items)
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error inesperado: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List() devolvió %d pedidos, se esperaba 1", len(list))
	}

	if err := repo.UpdateStatus(ctx, order.ID, domain.OrderStatusConfirmed); err != nil {
		t.Fatalf("UpdateStatus() error inesperado: %v", err)
	}
	updated, err := repo.GetByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetByID() tras update error inesperado: %v", err)
	}
	if updated.Status != domain.OrderStatusConfirmed {
		t.Errorf("Status = %q, se esperaba confirmed", updated.Status)
	}

	if err := repo.UpdateStatus(ctx, "00000000-0000-0000-0000-000000000000", domain.OrderStatusShipped); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("UpdateStatus() inexistente = %v, se esperaba ErrNotFound", err)
	}
	if _, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByID() inexistente = %v, se esperaba ErrNotFound", err)
	}
}

// TestIntegration_OrderFlow ejercita el flujo completo con repositorios reales:
// alta de catálogo, creación de pedido con descuento de stock, rollback ante
// stock insuficiente y restauración de stock al cancelar.
func TestIntegration_OrderFlow(t *testing.T) {
	truncateTables(t)
	ctx := testContext(t)

	categories := NewCategoryRepository(testPool)
	products := NewProductRepository(testPool)
	orders := NewOrderRepository(testPool)
	txManager := NewTxManager(testPool)

	productUC := usecase.NewProductUseCase(categories, products, txManager)
	orderUC := usecase.NewOrderUseCase(orders, products, txManager, "54 9 11 2233-4455")

	category, err := productUC.CreateCategory(ctx, usecase.CreateCategoryInput{Name: "Ropa"})
	if err != nil {
		t.Fatalf("CreateCategory() error inesperado: %v", err)
	}

	detail, err := productUC.CreateProduct(ctx, usecase.CreateProductInput{
		CategoryID: &category.ID,
		Name:       "Remera",
		IsActive:   true,
		Variants: []usecase.CreateVariantInput{
			{SKU: "REM-ROJO-S", Name: "Rojo / S", Price: 1500, StockQuantity: 5},
			{SKU: "REM-AZUL-S", Name: "Azul / S", Price: 1500, StockQuantity: 0},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct() error inesperado: %v", err)
	}

	var red, blue *domain.ProductVariant
	for i := range detail.Variants {
		switch detail.Variants[i].SKU {
		case "REM-ROJO-S":
			red = &detail.Variants[i]
		case "REM-AZUL-S":
			blue = &detail.Variants[i]
		}
	}
	if red == nil || blue == nil {
		t.Fatal("no se encontraron las variantes creadas")
	}

	result, err := orderUC.Create(ctx, usecase.CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "+5491122334455",
		Items:         []usecase.CreateOrderItemInput{{VariantID: red.ID, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("Create() pedido error inesperado: %v", err)
	}
	if result.Order.TotalAmount != 3000 {
		t.Errorf("TotalAmount = %v, se esperaba 3000", result.Order.TotalAmount)
	}
	if !strings.HasPrefix(result.WhatsAppLink, "https://wa.me/5491122334455?text=") {
		t.Errorf("WhatsAppLink inesperado: %q", result.WhatsAppLink)
	}

	afterSale, err := products.GetVariantByID(ctx, red.ID)
	if err != nil {
		t.Fatalf("GetVariantByID() error inesperado: %v", err)
	}
	if afterSale.StockQuantity != 3 {
		t.Fatalf("stock tras vender 2 = %d, se esperaba 3", afterSale.StockQuantity)
	}

	// El segundo item no tiene stock: la transacción debe revertir el descuento
	// del primero.
	_, err = orderUC.Create(ctx, usecase.CreateOrderInput{
		CustomerName:  "Juan",
		CustomerPhone: "+5491122334455",
		Items: []usecase.CreateOrderItemInput{
			{VariantID: red.ID, Quantity: 1},
			{VariantID: blue.ID, Quantity: 1},
		},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Create() sin stock = %v, se esperaba ErrValidation", err)
	}
	afterRollback, err := products.GetVariantByID(ctx, red.ID)
	if err != nil {
		t.Fatalf("GetVariantByID() error inesperado: %v", err)
	}
	if afterRollback.StockQuantity != 3 {
		t.Errorf("el rollback no restauró el stock: %d, se esperaba 3", afterRollback.StockQuantity)
	}

	persisted, err := orders.List(ctx)
	if err != nil {
		t.Fatalf("List() error inesperado: %v", err)
	}
	if len(persisted) != 1 {
		t.Fatalf("el pedido fallido no debe persistirse; hay %d pedidos", len(persisted))
	}

	if _, err := orderUC.UpdateStatus(ctx, result.Order.ID, domain.OrderStatusCancelled); err != nil {
		t.Fatalf("UpdateStatus() error inesperado: %v", err)
	}
	afterCancel, err := products.GetVariantByID(ctx, red.ID)
	if err != nil {
		t.Fatalf("GetVariantByID() error inesperado: %v", err)
	}
	if afterCancel.StockQuantity != 5 {
		t.Errorf("stock tras cancelar = %d, se esperaba 5", afterCancel.StockQuantity)
	}

	cancelled, err := orderUC.Get(ctx, result.Order.ID)
	if err != nil {
		t.Fatalf("Get() error inesperado: %v", err)
	}
	if cancelled.Order.Status != domain.OrderStatusCancelled {
		t.Errorf("Status = %q, se esperaba cancelled", cancelled.Order.Status)
	}
	if !strings.Contains(cancelled.WhatsAppText, "Remera") {
		t.Errorf("WhatsAppText no contiene el producto: %q", cancelled.WhatsAppText)
	}
}
