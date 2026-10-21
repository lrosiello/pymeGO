// Package logger expone un logger estructurado basado en log/slog, agnóstico
// al dominio de la aplicación.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New construye un *slog.Logger según el entorno. En producción emite JSON
// (ideal para agregadores de logs); en desarrollo emite texto legible.
func New(env string) *slog.Logger {
	level := slog.LevelDebug
	if strings.EqualFold(env, "production") {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.EqualFold(env, "production") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(slog.String("app", "catalogogo"))
}
