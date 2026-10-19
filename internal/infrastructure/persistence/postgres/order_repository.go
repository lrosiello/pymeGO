package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pymego/internal/domain"
)

// OrderRepository es la implementación PostgreSQL de domain.OrderRepository.
// Los items del pedido se persisten como snapshot en una columna JSONB,
// preservando el histórico aunque el catálogo cambie después.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository construye el repositorio de pedidos.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Verificación en tiempo de compilación de que se cumple el puerto del dominio.
var _ domain.OrderRepository = (*OrderRepository)(nil)

const orderColumns = `id, customer_name, customer_phone, total_amount, status, items, created_at, updated_at`

// Create inserta un pedido y rellena ID y timestamps del struct recibido.
func (r *OrderRepository) Create(ctx context.Context, order *domain.OrderLog) error {
	items, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("postgres: no se pudieron serializar los items del pedido: %w", err)
	}

	query := `INSERT INTO order_logs (customer_name, customer_phone, total_amount, status, items)
	          VALUES ($1, $2, $3, $4, $5)
	          RETURNING id, created_at, updated_at`

	err = querier(ctx, r.pool).QueryRow(ctx, query,
		order.CustomerName,
		order.CustomerPhone,
		order.TotalAmount,
		string(order.Status),
		items,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: error al crear pedido: %w", err)
	}

	return nil
}

// GetByID recupera un pedido por su identificador.
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*domain.OrderLog, error) {
	query := `SELECT ` + orderColumns + ` FROM order_logs WHERE id = $1`
	return scanOrder(querier(ctx, r.pool).QueryRow(ctx, query, id))
}

// List devuelve todos los pedidos ordenados por fecha de creación descendente.
func (r *OrderRepository) List(ctx context.Context) ([]domain.OrderLog, error) {
	query := `SELECT ` + orderColumns + ` FROM order_logs ORDER BY created_at DESC`

	rows, err := querier(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: error al listar pedidos: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.OrderLog, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: error al iterar pedidos: %w", err)
	}

	return orders, nil
}

// UpdateStatus cambia el estado de un pedido.
func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	query := `UPDATE order_logs
	          SET status = $2, updated_at = NOW()
	          WHERE id = $1
	          RETURNING updated_at`

	var updatedAt any
	err := querier(ctx, r.pool).QueryRow(ctx, query, id, string(status)).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("postgres: error al actualizar estado del pedido: %w", err)
	}

	return nil
}

// scanOrder materializa una fila en domain.OrderLog.
func scanOrder(row pgx.Row) (*domain.OrderLog, error) {
	var (
		order     domain.OrderLog
		rawStatus string
		rawItems  []byte
	)

	err := row.Scan(
		&order.ID,
		&order.CustomerName,
		&order.CustomerPhone,
		&order.TotalAmount,
		&rawStatus,
		&rawItems,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: error al leer pedido: %w", err)
	}

	order.Status = domain.OrderStatus(rawStatus)
	if len(rawItems) > 0 {
		if err := json.Unmarshal(rawItems, &order.Items); err != nil {
			return nil, fmt.Errorf("postgres: items del pedido con formato inválido: %w", err)
		}
	}

	return &order, nil
}
