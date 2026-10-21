package handler

import (
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"pymego/internal/delivery/http/dto"
	"pymego/internal/usecase"
)

// uuidPattern valida identificadores UUID en rutas y query params públicos.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// CatalogHandler expone la lectura pública de la vitrina.
type CatalogHandler struct {
	catalog usecase.CatalogUseCase
	log     *slog.Logger
}

// NewCatalogHandler construye el handler del catálogo público.
func NewCatalogHandler(catalog usecase.CatalogUseCase, log *slog.Logger) *CatalogHandler {
	return &CatalogHandler{catalog: catalog, log: log}
}

// ListCategories godoc
// @Summary  Lista las categorías activas
// @Tags     catalog
// @Produce  json
// @Success  200 {object} map[string][]dto.CatalogCategoryResponse
// @Failure  500 {object} errorResponse
// @Router   /api/v1/catalog/categories [get]
func (h *CatalogHandler) ListCategories(c *gin.Context) {
	categories, err := h.catalog.ListCategories(c.Request.Context())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	response := make([]dto.CatalogCategoryResponse, 0, len(categories))
	for i := range categories {
		response = append(response, dto.NewCatalogCategoryResponse(&categories[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// ListProducts godoc
// @Summary  Lista los productos vendibles con sus variantes
// @Tags     catalog
// @Produce  json
// @Param    category_id query string false "Filtra por categoría"
// @Success  200 {object} map[string][]dto.CatalogProductResponse
// @Failure  400,500 {object} errorResponse
// @Router   /api/v1/catalog/products [get]
func (h *CatalogHandler) ListProducts(c *gin.Context) {
	var categoryID *string
	if raw := strings.TrimSpace(c.Query("category_id")); raw != "" {
		if !isUUID(raw) {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "category_id inválido"})
			return
		}
		categoryID = &raw
	}

	products, err := h.catalog.ListProducts(c.Request.Context(), categoryID)
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	response := make([]dto.CatalogProductResponse, 0, len(products))
	for _, product := range products {
		response = append(response, dto.NewCatalogProductResponse(product))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// GetProduct godoc
// @Summary  Obtiene el detalle público de un producto
// @Tags     catalog
// @Produce  json
// @Param    id path string true "ID del producto"
// @Success  200 {object} dto.CatalogProductResponse
// @Failure  400,404,500 {object} errorResponse
// @Router   /api/v1/catalog/products/{id} [get]
func (h *CatalogHandler) GetProduct(c *gin.Context) {
	id := c.Param("id")
	if !isUUID(id) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "identificador inválido"})
		return
	}

	product, err := h.catalog.GetProduct(c.Request.Context(), id)
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewCatalogProductResponse(*product))
}

// isUUID indica si el valor tiene formato UUID.
func isUUID(value string) bool {
	return uuidPattern.MatchString(value)
}
