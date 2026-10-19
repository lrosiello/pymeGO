// TxManager es el puerto que abstrae la ejecución de operaciones de escritura
// dentro de una única transacción, sin acoplar la capa de casos de uso a pgx.
// La implementación de persistencia inyecta la transacción en el contexto y los
// repositorios la reutilizan de forma transparente.
package domain

import "context"

// TxManager ejecuta una función dentro de una transacción atómica.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
