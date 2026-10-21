package domain

import (
	"context"
	"time"
)

// OrderStatus representa el ciclo de vida de un pedido.
type OrderStatus string

// Estados de pedido soportados por el schema SQL.
const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// IsValid indica si el estado es uno de los permitidos.
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending, OrderStatusConfirmed, OrderStatusShipped, OrderStatusCancelled:
		return true
	default:
		return false
	}
}

// OrderItem es el snapshot de una línea del pedido. Se guarda desnormalizado
// dentro del campo JSONB "items" para preservar el histórico aunque el catálogo
// cambie después. Su forma coincide con la de whatsapp.Item.
type OrderItem struct {
	VariantID   string  `json:"variant_id"`
	ProductName string  `json:"product_name"`
	VariantName string  `json:"variant_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

// OrderLog registra el historial de un pedido realizado por un cliente vía
// WhatsApp. No referencia productos por FK: guarda un snapshot en Items.
type OrderLog struct {
	ID            string
	CustomerName  string
	CustomerPhone string
	TotalAmount   float64
	Status        OrderStatus
	Items         []OrderItem
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OrderRepository es el puerto de persistencia para pedidos.
type OrderRepository interface {
	Create(ctx context.Context, order *OrderLog) error
	GetByID(ctx context.Context, id string) (*OrderLog, error)
	List(ctx context.Context) ([]OrderLog, error)
	UpdateStatus(ctx context.Context, id string, status OrderStatus) error
}
