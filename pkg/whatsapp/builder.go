// Package whatsapp provee utilidades agnósticas al dominio para construir el
// texto y el enlace de WhatsApp de un pedido a partir de sus items. No importa
// la capa de dominio: define el contrato JSON que comparte con la columna
// "items" (JSONB) de order_logs.
package whatsapp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Item representa una línea del pedido tal como se persiste en el JSONBItems de
// la tabla order_logs. Su forma es el contrato compartido con la base de datos.
type Item struct {
	VariantID   string  `json:"variant_id"`
	ProductName string  `json:"product_name"`
	VariantName string  `json:"variant_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

// Subtotal devuelve unit_price * quantity.
func (i Item) Subtotal() float64 {
	return i.UnitPrice * float64(i.Quantity)
}

// BuildText arma el texto legible del pedido que se enviará por WhatsApp.
func BuildText(customerName string, items []Item) string {
	var b strings.Builder

	b.WriteString("¡Hola! Quiero confirmar el siguiente pedido:")
	b.WriteString("\n")
	if strings.TrimSpace(customerName) != "" {
		b.WriteString("\nCliente: ")
		b.WriteString(customerName)
		b.WriteString("\n")
	}

	var total float64
	for _, it := range items {
		line := it.ProductName
		if strings.TrimSpace(it.VariantName) != "" {
			line += " (" + it.VariantName + ")"
		}

		b.WriteString("\n• ")
		b.WriteString(strconv.Itoa(it.Quantity))
		b.WriteString(" x ")
		b.WriteString(line)
		b.WriteString(" — ")
		b.WriteString(formatMoney(it.UnitPrice))
		b.WriteString(" c/u")

		total += it.Subtotal()
	}

	b.WriteString("\n\nTotal: ")
	b.WriteString(formatMoney(total))

	return b.String()
}

// BuildLink arma la URL de wa.me con el número destino y el texto del pedido
// ya codificado. El número se normaliza dejando solo dígitos.
func BuildLink(phone, customerName string, items []Item) string {
	text := url.QueryEscape(BuildText(customerName, items))
	return fmt.Sprintf("https://wa.me/%s?text=%s", normalizePhone(phone), text)
}

// BuildLinkFromJSON deserializa los items desde su representación JSONB y arma
// el enlace. Útil al reconstruir el link desde un pedido ya persistido.
func BuildLinkFromJSON(phone, customerName string, rawItems []byte) (string, error) {
	var items []Item
	if err := json.Unmarshal(rawItems, &items); err != nil {
		return "", fmt.Errorf("whatsapp: items con formato inválido: %w", err)
	}
	return BuildLink(phone, customerName, items), nil
}

// normalizePhone deja únicamente los dígitos del número.
func normalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// formatMoney formatea un importe con dos decimales.
func formatMoney(amount float64) string {
	return "$" + strconv.FormatFloat(amount, 'f', 2, 64)
}
