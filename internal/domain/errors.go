// Package domain contiene las entidades de negocio puras, las reglas
// invariantes y las interfaces (puertos) que la capa de infraestructura debe
// implementar. No depende de frameworks ni de detalles de persistencia.
package domain

import "errors"

// Errores de dominio. Los casos de uso los retornan y la capa de delivery los
// traduce a códigos HTTP apropiados.
var (
	// ErrNotFound indica que el recurso solicitado no existe.
	ErrNotFound = errors.New("recurso no encontrado")
	// ErrConflict indica una violación de unicidad o estado duplicado.
	ErrConflict = errors.New("conflicto de recurso")
	// ErrUnauthorized indica credenciales inválidas o ausencia de permisos.
	ErrUnauthorized = errors.New("no autorizado")
	// ErrValidation indica que la entrada no cumple las reglas de negocio.
	ErrValidation = errors.New("error de validación")
)

// ValidationError describe un fallo de validación asociado a un campo concreto.
type ValidationError struct {
	Field   string
	Message string
}

// Error implementa la interfaz error.
func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Unwrap permite que errors.Is(err, ErrValidation) sea verdadero.
func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

// NewValidationError construye un ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}
