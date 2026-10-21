// Package middleware contiene middlewares de Gin transversales a los handlers.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pymego/pkg/jwt"
)

// Claves de contexto usadas para propagar la identidad autenticada.
const (
	ctxUserID = "auth_user_id"
	ctxRole   = "auth_role"
)

// JWTMiddleware valida el token Bearer e inyecta la identidad (user_id, role)
// en el contexto de Gin para su uso en capas posteriores.
func JWTMiddleware(manager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := extractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			abortUnauthorized(c, err.Error())
			return
		}

		claims, err := manager.Parse(token)
		if err != nil {
			abortUnauthorized(c, "token inválido o expirado")
			return
		}

		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxRole, claims.Role)

		c.Next()
	}
}

// extractBearerToken obtiene el token del encabezado Authorization.
func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("falta el encabezado Authorization")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("formato de Authorization inválido")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("token vacío")
	}
	return token, nil
}

// abortUnauthorized detiene la cadena y responde 401.
func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message})
}

// GetUserID devuelve el ID del usuario autenticado.
func GetUserID(c *gin.Context) string {
	return c.GetString(ctxUserID)
}

// GetRole devuelve el rol del usuario autenticado.
func GetRole(c *gin.Context) string {
	return c.GetString(ctxRole)
}

// RequireRole restringe el acceso a los roles indicados.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		if _, ok := allowed[GetRole(c)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "no tiene permisos para realizar esta acción",
			})
			return
		}
		c.Next()
	}
}
