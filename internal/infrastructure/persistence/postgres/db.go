// Package postgres implementa los puertos de persistencia definidos en el
// dominio usando pgx v5 sobre PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"pymego/internal/infrastructure/config"
)

// DBTX abstrae las operaciones comunes de consulta/ejecución, permitiendo que
// los repositorios trabajen indistintamente contra el pool o contra una
// transacción. Tanto *pgxpool.Pool como pgx.Tx satisfacen esta interfaz.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// txKey es el tipo de clave privada para inyectar la transacción en el contexto.
type txKey struct{}

// NewPool construye y verifica un pool de conexiones a PostgreSQL.
func NewPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: DATABASE_URL inválido: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: no se pudo crear el pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: no se pudo conectar a la base de datos: %w", err)
	}

	return pool, nil
}

// TxManager implementa la orquestación de transacciones sobre un pool de pgx.
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager crea un TxManager.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithinTransaction ejecuta fn dentro de una transacción. La transacción queda
// disponible en el contexto para que los repositorios la usen de forma
// transparente a través de querier(ctx).
func (m *TxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: no se pudo iniciar la transacción: %w", err)
	}

	defer func() {
		// Rollback es no-op si la transacción ya fue confirmada.
		_ = tx.Rollback(ctx)
	}()

	txCtx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(txCtx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: no se pudo confirmar la transacción: %w", err)
	}

	return nil
}

// querier devuelve la transacción activa del contexto si existe; en caso
// contrario, el pool de conexiones. Es el punto único de resolución de la
// conexión para todos los repositorios.
func querier(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// isUniqueViolation indica si el error proviene de una violación de unicidad de
// PostgreSQL (código SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
