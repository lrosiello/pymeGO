package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"pymego/internal/domain"
	"pymego/pkg/whatsapp"
)

// OrderUseCase define las operaciones del módulo de pedidos: registro con
// descuento atómico de stock, seguimiento y generación del enlace de WhatsApp.
type OrderUseCase interface {
	// Create registra un pedido del cliente, descuenta el stock de cada variante
	// de forma atómica y devuelve el enlace de WhatsApp listo para enviar.
	Create(ctx context.Context, input CreateOrderInput) (*OrderResult, error)
	// List devuelve todos los pedidos.
	List(ctx context.Context) ([]domain.OrderLog, error)
	// Get devuelve un pedido con el enlace de WhatsApp regenerado.
	Get(ctx context.Context, orderID string) (*OrderResult, error)
	// UpdateStatus cambia el estado del pedido; al cancelar, devuelve el stock.
	UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus) (*domain.OrderLog, error)
}

// CreateOrderInput son los datos para registrar un pedido.
type CreateOrderInput struct {
	CustomerName  string
	CustomerPhone string
	Items         []CreateOrderItemInput
}

// CreateOrderItemInput es una línea del pedido: variante y cantidad. El precio
// y el nombre se resuelven del servidor (nunca se confía en el cliente).
type CreateOrderItemInput struct {
	VariantID string
	Quantity  int
}

// OrderResult agrupa el pedido persistido con su mensaje/enlace de WhatsApp.
type OrderResult struct {
	Order        *domain.OrderLog
	WhatsAppText string
	WhatsAppLink string
}

// orderUseCase implementa OrderUseCase.
type orderUseCase struct {
	orders         domain.OrderRepository
	products       domain.ProductRepository
	tx             domain.TxManager
	whatsappNumber string
}

// NewOrderUseCase construye el caso de uso de pedidos. whatsappNumber es el
// número destino del comercio (configuración) usado para armar el enlace.
func NewOrderUseCase(
	orders domain.OrderRepository,
	products domain.ProductRepository,
	tx domain.TxManager,
	whatsappNumber string,
) OrderUseCase {
	return &orderUseCase{
		orders:         orders,
		products:       products,
		tx:             tx,
		whatsappNumber: whatsappNumber,
	}
}

// Create valida la entrada, construye el snapshot de items resolviendo precios y
// nombres desde el catálogo, descuenta el stock atómicamente y persiste el
// pedido. Todo ocurre dentro de una única transacción: si una variante no tiene
// stock suficiente, no queda ningún cambio aplicado.
func (uc *orderUseCase) Create(ctx context.Context, input CreateOrderInput) (*OrderResult, error) {
	customerName := strings.TrimSpace(input.CustomerName)
	if customerName == "" {
		return nil, domain.NewValidationError("customer_name", "es obligatorio")
	}
	customerPhone := strings.TrimSpace(input.CustomerPhone)
	if customerPhone == "" {
		return nil, domain.NewValidationError("customer_phone", "es obligatorio")
	}
	if len(input.Items) == 0 {
		return nil, domain.NewValidationError("items", "el pedido debe tener al menos una línea")
	}

	order := &domain.OrderLog{
		CustomerName:  customerName,
		CustomerPhone: customerPhone,
		Status:        domain.OrderStatusPending,
		Items:         make([]domain.OrderItem, 0, len(input.Items)),
	}

	var total float64

	err := uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		for i, in := range input.Items {
			if in.Quantity <= 0 {
				return fmt.Errorf("items[%d]: %w", i, domain.NewValidationError("quantity", "debe ser mayor que cero"))
			}

			variant, err := uc.products.GetVariantByID(txCtx, in.VariantID)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					return fmt.Errorf("items[%d]: %w", i, domain.NewValidationError("variant_id", "la variante no existe"))
				}
				return err
			}
			if !variant.IsActive {
				return fmt.Errorf("items[%d]: %w", i, domain.NewValidationError("variant_id", "la variante está inactiva"))
			}

			product, err := uc.products.GetByID(txCtx, variant.ProductID)
			if err != nil {
				return err
			}

			// Descuento atómico: falla con error de validación si no hay stock.
			if _, err := uc.products.AdjustStockQuantity(txCtx, in.VariantID, -in.Quantity); err != nil {
				if errors.Is(err, domain.ErrValidation) {
					return fmt.Errorf("items[%d]: %w", i, err)
				}
				return err
			}

			order.Items = append(order.Items, domain.OrderItem{
				VariantID:   variant.ID,
				ProductName: product.Name,
				VariantName: variant.Name,
				Quantity:    in.Quantity,
				UnitPrice:   variant.Price,
			})
			total += variant.Price * float64(in.Quantity)
		}

		order.TotalAmount = total
		return uc.orders.Create(txCtx, order)
	})
	if err != nil {
		return nil, err
	}

	return uc.buildResult(order), nil
}

// List devuelve todos los pedidos.
func (uc *orderUseCase) List(ctx context.Context) ([]domain.OrderLog, error) {
	return uc.orders.List(ctx)
}

// Get devuelve un pedido junto con su enlace de WhatsApp regenerado.
func (uc *orderUseCase) Get(ctx context.Context, orderID string) (*OrderResult, error) {
	order, err := uc.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return uc.buildResult(order), nil
}

// UpdateStatus cambia el estado del pedido. Si se cancela un pedido que no
// estaba cancelado, devuelve el stock de sus variantes de forma atómica.
func (uc *orderUseCase) UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus) (*domain.OrderLog, error) {
	if !status.IsValid() {
		return nil, domain.NewValidationError("status", "debe ser uno de: pending, confirmed, shipped, cancelled")
	}

	order, err := uc.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status == status {
		return order, nil
	}

	restoreStock := status == domain.OrderStatusCancelled && order.Status != domain.OrderStatusCancelled

	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if restoreStock {
			for _, item := range order.Items {
				if _, err := uc.products.AdjustStockQuantity(txCtx, item.VariantID, item.Quantity); err != nil {
					// Si la variante ya no existe, no bloqueamos la cancelación.
					if errors.Is(err, domain.ErrNotFound) {
						continue
					}
					return err
				}
			}
		}
		return uc.orders.UpdateStatus(txCtx, orderID, status)
	})
	if err != nil {
		return nil, err
	}

	order.Status = status
	return order, nil
}

// buildResult arma el texto y el enlace de WhatsApp a partir del pedido.
func (uc *orderUseCase) buildResult(order *domain.OrderLog) *OrderResult {
	items := toWhatsAppItems(order.Items)

	text := whatsapp.BuildText(order.CustomerName, items)
	link := whatsapp.BuildLink(uc.whatsappNumber, order.CustomerName, items)

	return &OrderResult{
		Order:        order,
		WhatsAppText: text,
		WhatsAppLink: link,
	}
}

// toWhatsAppItems traduce el snapshot del pedido al contrato del helper de
// WhatsApp.
func toWhatsAppItems(items []domain.OrderItem) []whatsapp.Item {
	out := make([]whatsapp.Item, 0, len(items))
	for _, it := range items {
		out = append(out, whatsapp.Item{
			VariantID:   it.VariantID,
			ProductName: it.ProductName,
			VariantName: it.VariantName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
		})
	}
	return out
}
