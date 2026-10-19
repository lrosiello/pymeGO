// Package handler contiene los controladores HTTP (Gin). Su responsabilidad es
// traducir la petición al input del caso de uso, y el resultado/errores del
// dominio a respuestas HTTP con el código de estado adecuado.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"pymego/internal/delivery/http/dto"
	"pymego/internal/delivery/http/middleware"
	"pymego/internal/usecase"
)

// AuthHandler expone los endpoints de autenticación.
type AuthHandler struct {
	auth usecase.AuthUseCase
	log  *slog.Logger
}

// NewAuthHandler construye el handler de autenticación.
func NewAuthHandler(auth usecase.AuthUseCase, log *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, log: log}
}

// Register godoc
// @Summary  Registra al administrador del comercio
// @Tags     auth
// @Accept   json
// @Produce  json
// @Success  201 {object} dto.AuthResponse
// @Failure  400,409,422,500 {object} errorResponse
// @Router   /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	result, err := h.auth.Register(c.Request.Context(), req.ToInput())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusCreated, dto.AuthResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      dto.NewUserResponse(result.User),
	})
}

// Login godoc
// @Summary  Autentica un usuario y emite un token JWT
// @Tags     auth
// @Accept   json
// @Produce  json
// @Success  200 {object} dto.AuthResponse
// @Failure  400,401,500 {object} errorResponse
// @Router   /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, buildValidationError(err))
		return
	}

	result, err := h.auth.Login(c.Request.Context(), req.ToInput())
	if err != nil {
		respondDomainError(c, h.log, err)
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      dto.NewUserResponse(result.User),
	})
}

// Me devuelve la identidad extraída del token por el JWTMiddleware. Sirve como
// endpoint de verificación de sesión.
func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"user_id": middleware.GetUserID(c),
		"role":    middleware.GetRole(c),
	})
}
