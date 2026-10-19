package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"pymego/internal/domain"
)

// errorResponse es el formato uniforme de error de la API, compartido por todos
// los handlers del paquete.
type errorResponse struct {
	Error   string            `json:"error"`
	Fields  map[string]string `json:"fields,omitempty"`
	Details string            `json:"details,omitempty"`
}

// respondDomainError traduce los errores del dominio a respuestas HTTP con el
// código de estado adecuado. Los errores no reconocidos se registran y se
// exponen como 500 sin filtrar detalles internos.
func respondDomainError(c *gin.Context, log *slog.Logger, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		var vErr *domain.ValidationError
		if errors.As(err, &vErr) {
			c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Error:  "error de validación",
				Fields: map[string]string{vErr.Field: vErr.Message},
			})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Error: "error de validación"})

	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "credenciales inválidas"})

	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: "recurso no encontrado"})

	default:
		log.Error("error inesperado", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "error interno del servidor"})
	}
}

// buildValidationError transforma los errores de binding/validator en un
// errorResponse con el detalle por campo.
func buildValidationError(err error) errorResponse {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		fields := make(map[string]string, len(verrs))
		for _, fe := range verrs {
			fields[fe.Field()] = validationMessage(fe)
		}
		return errorResponse{Error: "error de validación", Fields: fields}
	}
	return errorResponse{Error: "cuerpo de la petición inválido", Details: err.Error()}
}

// validationMessage traduce una etiqueta de validación a un mensaje en español.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "es obligatorio"
	case "email":
		return "debe ser un email válido"
	case "min":
		return "debe tener al menos " + fe.Param() + " caracteres"
	case "max":
		return "debe tener como máximo " + fe.Param() + " caracteres"
	case "gte":
		return "debe ser mayor o igual a " + fe.Param()
	case "lte":
		return "debe ser menor o igual a " + fe.Param()
	case "ne":
		return "no puede ser " + fe.Param()
	case "oneof":
		return "debe ser uno de: " + fe.Param()
	default:
		return "valor inválido"
	}
}
