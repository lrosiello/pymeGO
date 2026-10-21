// Package router configura el enrutador Gin, los middlewares globales y el
// árbol de rutas de la API.
package router

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pymego/internal/delivery/http/handler"
	"pymego/internal/delivery/http/middleware"
	"pymego/internal/infrastructure/config"
	"pymego/pkg/jwt"
)

// New construye el *gin.Engine con las rutas públicas y protegidas.
func New(
	cfg *config.Config,
	log *slog.Logger,
	authHandler *handler.AuthHandler,
	productHandler *handler.ProductHandler,
	orderHandler *handler.OrderHandler,
	jwtManager *jwt.Manager,
) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger(log))
	r.Use(corsMiddleware(cfg.App.CORSOrigins))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")

	// Rutas públicas.
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Checkout público: el cliente registra su pedido y recibe el link de
	// WhatsApp. El precio y el stock se resuelven en el servidor.
	v1.POST("/orders", orderHandler.Create)

	// Rutas protegidas por JWT.
	protected := v1.Group("")
	protected.Use(middleware.JWTMiddleware(jwtManager))
	{
		protected.GET("/me", authHandler.Me)
	}

	// Catálogo: solo administradores y staff del comercio.
	catalog := protected.Group("")
	catalog.Use(middleware.RequireRole("admin", "staff"))
	{
		catalog.POST("/categories", productHandler.CreateCategory)
		catalog.GET("/categories", productHandler.ListCategories)
		catalog.PUT("/categories/:id", productHandler.UpdateCategory)
		catalog.DELETE("/categories/:id", productHandler.DeleteCategory)

		catalog.POST("/products", productHandler.CreateProduct)
		catalog.GET("/products", productHandler.ListProducts)
		catalog.GET("/products/:id", productHandler.GetProduct)
		catalog.PUT("/products/:id", productHandler.UpdateProduct)
		catalog.DELETE("/products/:id", productHandler.DeleteProduct)

		catalog.GET("/products/:id/variants", productHandler.ListVariants)
		catalog.POST("/products/:id/variants", productHandler.AddVariant)
		catalog.PUT("/variants/:id", productHandler.UpdateVariant)
		catalog.DELETE("/variants/:id", productHandler.DeleteVariant)
		catalog.PATCH("/variants/:id/stock", productHandler.AdjustStock)

		// Gestión de pedidos (solo admin/staff).
		catalog.GET("/orders", orderHandler.List)
		catalog.GET("/orders/:id", orderHandler.Get)
		catalog.PATCH("/orders/:id/status", orderHandler.UpdateStatus)
	}

	return r
}

// requestLogger emite una línea estructurada por petición usando slog.
func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log.Info("http_request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

// corsMiddleware aplica una política CORS básica basada en la configuración.
func corsMiddleware(origins []string) gin.HandlerFunc {
	allowAll := len(origins) == 0 || (len(origins) == 1 && origins[0] == "*")
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if allowAll {
				c.Header("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if strings.EqualFold(c.Request.Method, http.MethodOptions) {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
