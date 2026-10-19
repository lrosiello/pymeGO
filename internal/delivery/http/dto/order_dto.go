package dto

import (
	"time"

	"pymego/internal/domain"
	"pymego/internal/usecase"
)

// CreateOrderRequest es el cuerpo de POST /api/v1/orders.
type CreateOrderRequest struct {
	CustomerName  string                   `json:"customer_name" binding:"required,min=2,max=150"`
	CustomerPhone string                   `json:"customer_phone" binding:"required,max=50"`
	Items         []CreateOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

// CreateOrderItemRequest es una línea del pedido.
type CreateOrderItemRequest struct {
	VariantID string `json:"variant_id" binding:"required,uuid4"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
}

// ToInput traduce el DTO al input del caso de uso.
func (r CreateOrderRequest) ToInput() usecase.CreateOrderInput {
	items := make([]usecase.CreateOrderItemInput, 0, len(r.Items))
	for _, it := range r.Items {
		items = append(items, usecase.CreateOrderItemInput{
			VariantID: it.VariantID,
			Quantity:  it.Quantity,
		})
	}
	return usecase.CreateOrderInput{
		CustomerName:  r.CustomerName,
		CustomerPhone: r.CustomerPhone,
		Items:         items,
	}
}

// UpdateOrderStatusRequest es el cuerpo de PATCH /api/v1/orders/:id/status.
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending confirmed shipped cancelled"`
}

// ToStatus traduce el DTO al tipo de dominio.
func (r UpdateOrderStatusRequest) ToStatus() domain.OrderStatus {
	return domain.OrderStatus(r.Status)
}

// OrderItemResponse es la representación pública de una línea del pedido.
type OrderItemResponse struct {
	VariantID   string  `json:"variant_id"`
	ProductName string  `json:"product_name"`
	VariantName string  `json:"variant_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"`
}

// OrderResponse es la representación pública de un pedido. Incluye el enlace y
// el texto de WhatsApp cuando están disponibles.
type OrderResponse struct {
	ID            string              `json:"id"`
	CustomerName  string              `json:"customer_name"`
	CustomerPhone string              `json:"customer_phone"`
	TotalAmount   float64             `json:"total_amount"`
	Status        string              `json:"status"`
	Items         []OrderItemResponse `json:"items"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	WhatsAppText  string              `json:"whatsapp_text,omitempty"`
	WhatsAppLink  string              `json:"whatsapp_link,omitempty"`
}

// NewOrderResponse construye la respuesta desde el pedido persistido.
func NewOrderResponse(order *domain.OrderLog) OrderResponse {
	items := make([]OrderItemResponse, 0, len(order.Items))
	for _, it := range order.Items {
		items = append(items, OrderItemResponse{
			VariantID:   it.VariantID,
			ProductName: it.ProductName,
			VariantName: it.VariantName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
			Subtotal:    it.UnitPrice * float64(it.Quantity),
		})
	}

	return OrderResponse{
		ID:            order.ID,
		CustomerName:  order.CustomerName,
		CustomerPhone: order.CustomerPhone,
		TotalAmount:   order.TotalAmount,
		Status:        string(order.Status),
		Items:         items,
		CreatedAt:     order.CreatedAt,
		UpdatedAt:     order.UpdatedAt,
	}
}

// NewOrderResultResponse construye la respuesta incluyendo el enlace y el texto
// de WhatsApp generados por el caso de uso.
func NewOrderResultResponse(result usecase.OrderResult) OrderResponse {
	response := NewOrderResponse(result.Order)
	response.WhatsAppText = result.WhatsAppText
	response.WhatsAppLink = result.WhatsAppLink
	return response
}
