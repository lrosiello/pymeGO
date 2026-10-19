package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"pymego/internal/delivery/http/dto"
	"pymego/internal/usecase"
)

// ProductHandler expone los endpoints del catálogo: categorías, productos y
// variantes (incluido el ajuste de stock atómico).
type ProductHandler struct {
	products usecase.ProductUseCase
	log      *slog.Logger
}

// NewProductHandler construye el handler de catálogo.
func NewProductHandler(products usecase.ProductUseCase, log *slog.Logger) *ProductHandler {
	return &ProductHandler{products: products, log: log}
}

// ---------------------------------------------------------------------------
// Categorías
// ---------------------------------------------------------------------------

// CreateCategory godoc
// @Summary  Crea una categoría
// @Tags     categories
// @Accept   json
// @Produce  json
// @Success  201 {object} dto.CategoryResponse
// @Failure  400,422,500 {object} errorResponse
// @Router   /api/v1/categories [post]
func (h *ProductHandler) CreateCategory(c *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	category, err := h.products.CreateCategory(c.Request.Context(), req.ToInput())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusCreated, dto.NewCategoryResponse(category))
}

// ListCategories godoc
// @Summary  Lista las categorías
// @Tags     categories
// @Produce  json
// @Success  200 {object} map[string][]dto.CategoryResponse
// @Failure  500 {object} errorResponse
// @Router   /api/v1/categories [get]
func (h *ProductHandler) ListCategories(c *gin.Context) {
	categories, err := h.products.ListCategories(c.Request.Context())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	response := make([]dto.CategoryResponse, 0, len(categories))
	for i := range categories {
		response = append(response, dto.NewCategoryResponse(&categories[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// UpdateCategory godoc
// @Summary  Actualiza una categoría
// @Tags     categories
// @Accept   json
// @Produce  json
// @Success  200 {object} dto.CategoryResponse
// @Failure  400,404,422,500 {object} errorResponse
// @Router   /api/v1/categories/{id} [put]
func (h *ProductHandler) UpdateCategory(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	category, err := h.products.UpdateCategory(c.Request.Context(), id, req.ToInput())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewCategoryResponse(category))
}

// DeleteCategory godoc
// @Summary  Elimina una categoría
// @Tags     categories
// @Produce  json
// @Success  204
// @Failure  404,500 {object} errorResponse
// @Router   /api/v1/categories/{id} [delete]
func (h *ProductHandler) DeleteCategory(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	if err := h.products.DeleteCategory(c.Request.Context(), id); err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Productos
// ---------------------------------------------------------------------------

// CreateProduct godoc
// @Summary  Crea un producto (opcionalmente con variantes, de forma atómica)
// @Tags     products
// @Accept   json
// @Produce  json
// @Success  201 {object} dto.ProductResponse
// @Failure  400,409,422,500 {object} errorResponse
// @Router   /api/v1/products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	detail, err := h.products.CreateProduct(c.Request.Context(), req.ToInput())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusCreated, dto.NewProductResponse(*detail))
}

// ListProducts godoc
// @Summary  Lista los productos con sus variantes
// @Tags     products
// @Produce  json
// @Success  200 {object} map[string][]dto.ProductResponse
// @Failure  500 {object} errorResponse
// @Router   /api/v1/products [get]
func (h *ProductHandler) ListProducts(c *gin.Context) {
	details, err := h.products.ListProducts(c.Request.Context())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	response := make([]dto.ProductResponse, 0, len(details))
	for _, detail := range details {
		response = append(response, dto.NewProductResponse(detail))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// GetProduct godoc
// @Summary  Obtiene un producto con sus variantes
// @Tags     products
// @Produce  json
// @Success  200 {object} dto.ProductResponse
// @Failure  404,500 {object} errorResponse
// @Router   /api/v1/products/{id} [get]
func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	detail, err := h.products.GetProduct(c.Request.Context(), id)
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewProductResponse(*detail))
}

// UpdateProduct godoc
// @Summary  Actualiza un producto
// @Tags     products
// @Accept   json
// @Produce  json
// @Success  200 {object} dto.ProductResponse
// @Failure  400,404,422,500 {object} errorResponse
// @Router   /api/v1/products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	product, err := h.products.UpdateProduct(c.Request.Context(), id, req.ToInput())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewProductResponse(usecase.ProductDetail{Product: product}))
}

// DeleteProduct godoc
// @Summary  Elimina un producto y sus variantes
// @Tags     products
// @Produce  json
// @Success  204
// @Failure  404,500 {object} errorResponse
// @Router   /api/v1/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	if err := h.products.DeleteProduct(c.Request.Context(), id); err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Variantes
// ---------------------------------------------------------------------------

// ListVariants godoc
// @Summary  Lista las variantes de un producto
// @Tags     variants
// @Produce  json
// @Success  200 {object} map[string][]dto.VariantResponse
// @Failure  404,500 {object} errorResponse
// @Router   /api/v1/products/{id}/variants [get]
func (h *ProductHandler) ListVariants(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	variants, err := h.products.ListVariants(c.Request.Context(), id)
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	response := make([]dto.VariantResponse, 0, len(variants))
	for i := range variants {
		response = append(response, dto.NewVariantResponse(&variants[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// AddVariant godoc
// @Summary  Agrega una variante a un producto
// @Tags     variants
// @Accept   json
// @Produce  json
// @Success  201 {object} dto.VariantResponse
// @Failure  400,404,409,422,500 {object} errorResponse
// @Router   /api/v1/products/{id}/variants [post]
func (h *ProductHandler) AddVariant(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req dto.CreateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	variant, err := h.products.AddVariant(c.Request.Context(), id, req.ToInput())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusCreated, dto.NewVariantResponse(variant))
}

// UpdateVariant godoc
// @Summary  Actualiza una variante
// @Tags     variants
// @Accept   json
// @Produce  json
// @Success  200 {object} dto.VariantResponse
// @Failure  400,404,409,422,500 {object} errorResponse
// @Router   /api/v1/variants/{id} [put]
func (h *ProductHandler) UpdateVariant(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req dto.UpdateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	variant, err := h.products.UpdateVariant(c.Request.Context(), id, req.ToInput())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewVariantResponse(variant))
}

// DeleteVariant godoc
// @Summary  Elimina una variante
// @Tags     variants
// @Produce  json
// @Success  204
// @Failure  404,500 {object} errorResponse
// @Router   /api/v1/variants/{id} [delete]
func (h *ProductHandler) DeleteVariant(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	if err := h.products.DeleteVariant(c.Request.Context(), id); err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// AdjustStock godoc
// @Summary  Ajusta el stock de una variante de forma atómica
// @Tags     variants
// @Accept   json
// @Produce  json
// @Success  200 {object} dto.VariantResponse
// @Failure  400,404,422,500 {object} errorResponse
// @Router   /api/v1/variants/{id}/stock [patch]
func (h *ProductHandler) AdjustStock(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req dto.AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	variant, err := h.products.AdjustStock(c.Request.Context(), id, req.Delta)
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewVariantResponse(variant))
}

// pathID extrae el parámetro :id de la ruta y responde 400 si viene vacío.
func pathID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "el identificador es obligatorio"})
		return "", false
	}
	return id, true
}
