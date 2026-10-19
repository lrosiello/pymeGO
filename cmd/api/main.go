// Package main es el punto de entrada de CatálogoGO. Realiza la carga de
// configuración, la conexión a PostgreSQL y la inyección de dependencias
// siguiendo el flujo de Clean Architecture (infraestructura -> casos de uso ->
// delivery).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pymego/internal/delivery/http/handler"
	"pymego/internal/delivery/http/router"
	"pymego/internal/infrastructure/config"
	"pymego/internal/infrastructure/persistence/postgres"
	"pymego/internal/usecase"
	"pymego/pkg/jwt"
	"pymego/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fallo al iniciar la aplicación", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// run concentra el arranque para permitir un manejo de errores limpio y el
// cierre ordenado (graceful shutdown).
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.App.Env)
	log.Info("iniciando CatálogoGO", slog.String("env", cfg.App.Env))

	// Contexto cancelado ante SIGINT/SIGTERM para propagar el apagado.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Infraestructura: pool de conexiones.
	pool, err := postgres.NewPool(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("conexión a PostgreSQL establecida")

	// Repositorios.
	userRepo := postgres.NewUserRepository(pool)
	categoryRepo := postgres.NewCategoryRepository(pool)
	productRepo := postgres.NewProductRepository(pool)
	orderRepo := postgres.NewOrderRepository(pool)
	txManager := postgres.NewTxManager(pool)

	// Servicios.
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.ExpiresIn)

	// Casos de uso.
	authUseCase := usecase.NewAuthUseCase(userRepo, jwtManager)
	productUseCase := usecase.NewProductUseCase(categoryRepo, productRepo, txManager)
	orderUseCase := usecase.NewOrderUseCase(orderRepo, productRepo, txManager, cfg.WhatsApp.Number)

	// Handlers.
	authHandler := handler.NewAuthHandler(authUseCase, log)
	productHandler := handler.NewProductHandler(productUseCase, log)
	orderHandler := handler.NewOrderHandler(orderUseCase, log)

	// Router.
	engine := router.New(cfg, log, authHandler, productHandler, orderHandler, jwtManager)

	server := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      engine,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
	}

	// Arranque del servidor en segundo plano.
	serverErr := make(chan error, 1)
	go func() {
		log.Info("servidor HTTP escuchando", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Espera de señal de parada o error fatal del servidor.
	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Info("señal de apagado recibida, cerrando el servidor")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Info("servidor detenido correctamente")
	return nil
}
