// Package config centraliza la carga y validación de la configuración de la
// aplicación a partir de variables de entorno (con soporte opcional de archivo
// .env en desarrollo).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"pymego/internal/domain"
)

// Config agrupa toda la configuración tipada de la aplicación.
type Config struct {
	App      AppConfig
	DB       DBConfig
	JWT      JWTConfig
	WhatsApp WhatsAppConfig
	Catalog  CatalogConfig
}

// AppConfig contiene parámetros generales del servidor HTTP.
type AppConfig struct {
	Env          string
	Port         string
	CORSOrigins  []string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DBConfig contiene los parámetros de conexión a PostgreSQL.
type DBConfig struct {
	DatabaseURL     string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

// JWTConfig contiene los parámetros de emisión y validación de tokens.
type JWTConfig struct {
	Secret     string
	Issuer     string
	ExpiresIn  time.Duration
	SignMethod string
}

// WhatsAppConfig contiene el número destino del comercio.
type WhatsAppConfig struct {
	Number string
}

// CatalogConfig contiene la política de precios del comercio.
type CatalogConfig struct {
	PriceMode domain.PriceMode
}

// IsProduction indica si la aplicación corre en entorno productivo.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.App.Env, "production")
}

// Load carga la configuración desde el entorno. Si existe un archivo .env en
// el directorio de trabajo, sus valores se cargan sin sobrescribir variables
// de entorno ya presentes. Retorna error si falta alguna variable obligatoria
// o si algún valor no puede interpretarse.
func Load() (*Config, error) {
	// El archivo .env es opcional: en producción se usan variables reales.
	_ = godotenv.Load()

	expiresIn, err := time.ParseDuration(envOrDefault("JWT_EXPIRES_IN", "24h"))
	if err != nil {
		return nil, fmt.Errorf("config: JWT_EXPIRES_IN inválido: %w", err)
	}

	priceMode, err := domain.ParsePriceMode(envOrDefault("CATALOG_PRICE_MODE", string(domain.PriceModeB2C)))
	if err != nil {
		return nil, fmt.Errorf("config: CATALOG_PRICE_MODE inválido: %w", err)
	}

	cfg := &Config{
		App: AppConfig{
			Env:          envOrDefault("APP_ENV", "development"),
			Port:         envOrDefault("APP_PORT", "8080"),
			CORSOrigins:  splitAndTrim(envOrDefault("CORS_ORIGINS", "*")),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		DB: DBConfig{
			DatabaseURL:     os.Getenv("DATABASE_URL"),
			MaxConns:        int32(envInt("DB_MAX_CONNS", 10)),
			MinConns:        int32(envInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: time.Hour,
		},
		JWT: JWTConfig{
			Secret:     os.Getenv("JWT_SECRET"),
			Issuer:     envOrDefault("JWT_ISSUER", "catalogogo"),
			ExpiresIn:  expiresIn,
			SignMethod: "HS256",
		},
		WhatsApp: WhatsAppConfig{
			Number: envOrDefault("WHATSAPP_NUMBER", ""),
		},
		Catalog: CatalogConfig{
			PriceMode: priceMode,
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate comprueba que la configuración mínima esté presente y sea coherente.
func (c *Config) validate() error {
	if c.DB.DatabaseURL == "" {
		return fmt.Errorf("config: la variable DATABASE_URL es obligatoria")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("config: la variable JWT_SECRET es obligatoria")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("config: JWT_SECRET debe tener al menos 32 caracteres")
	}
	if c.DB.MinConns > c.DB.MaxConns {
		return fmt.Errorf("config: DB_MIN_CONNS no puede ser mayor que DB_MAX_CONNS")
	}
	return nil
}

// envOrDefault devuelve la variable de entorno o un valor por defecto.
func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// envInt interpreta una variable de entorno como entero, con valor por defecto.
func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// splitAndTrim separa por comas y limpia espacios de cada elemento.
func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
