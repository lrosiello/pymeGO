package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"pymego/internal/delivery/http/dto"
	"pymego/internal/usecase"
)

// OrderHandler expone los endpoints de pedidos: registro con descuento atómico
// de stock, listado, consulta con enlace de WhatsApp y cambio de estado.
type OrderHandler struct {
	orders usecase.OrderUseCase
	log    *slog.Logger
}

// NewOrderHandler construye el handler de pedidos.
func NewOrderHandler(orders usecase.OrderUseCase, log *slog.Logger) *OrderHandler {
	return &OrderHandler{orders: orders, log: log}
}

// Create godoc
// @Summary  Registra un pedido y devuelve el enlace de WhatsApp
// @Tags     orders
// @Accept   json
// @Produce  json
// @Success  201 {object} dto.OrderResponse
// @Failure  400,404,422,500 {object} errorResponse
// @Router   /api/v1/orders [post]
func (h *OrderHandler) Create(c *gin.Context) {
	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	result, err := h.orders.Create(c.Request.Context(), req.ToInput())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusCreated, dto.NewOrderResultResponse(*result))
}

// List godoc
// @Summary  Lista todos los pedidos
// @Tags     orders
// @Produce  json
// @Success  200 {object} map[string][]dto.OrderResponse
// @Failure  500 {object} errorResponse
// @Router   /api/v1/orders [get]
func (h *OrderHandler) List(c *gin.Context) {
	orders, err := h.orders.List(c.Request.Context())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	response := make([]dto.OrderResponse, 0, len(orders))
	for i := range orders {
		response = append(response, dto.NewOrderResponse(&orders[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// Get godoc
// @Summary  Obtiene un pedido con su enlace de WhatsApp
// @Tags     orders
// @Produce  json
// @Success  200 {object} dto.OrderResponse
// @Failure  404,500 {object} errorResponse
// @Router   /api/v1/orders/{id} [get]
func (h *OrderHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	result, err := h.orders.Get(c.Request.Context(), id)
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewOrderResultResponse(*result))
}

// UpdateStatus godoc
// @Summary  Actualiza el estado de un pedido (al cancelar, restaura el stock)
// @Tags     orders
// @Accept   json
// @Produce  json
// @Success  200 {object} dto.OrderResponse
// @Failure  400,404,422,500 {object} errorResponse
// @Router   /api/v1/orders/{id}/status [patch]
func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req dto.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	order, err := h.orders.UpdateStatus(c.Request.Context(), id, req.ToStatus())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewOrderResponse(order))
}
