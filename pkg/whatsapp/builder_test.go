package whatsapp

import (
	"net/url"
	"strings"
	"testing"
)

func sampleItems() []Item {
	return []Item{
		{VariantID: "v1", ProductName: "Remera", VariantName: "Rojo / S", Quantity: 2, UnitPrice: 1500},
		{VariantID: "v2", ProductName: "Pantalón", VariantName: "Azul / 42", Quantity: 1, UnitPrice: 3000},
	}
}

func TestSubtotal(t *testing.T) {
	item := Item{Quantity: 3, UnitPrice: 1000.5}
	if got, want := item.Subtotal(), 3001.5; got != want {
		t.Fatalf("Subtotal() = %v, se esperaba %v", got, want)
	}
}

func TestBuildText(t *testing.T) {
	text := BuildText("Juan Pérez", sampleItems())

	wants := []string{
		"Cliente: Juan Pérez",
		"2 x Remera (Rojo / S)",
		"1 x Pantalón (Azul / 42)",
		"$1500.00 c/u",
		"Total: $6000.00",
	}
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("BuildText() no contiene %q\n--- texto ---\n%s", want, text)
		}
	}
}

func TestBuildText_SinNombreCliente(t *testing.T) {
	text := BuildText("   ", sampleItems())
	if strings.Contains(text, "Cliente:") {
		t.Errorf("BuildText() no debería incluir 'Cliente:' cuando el nombre está vacío; texto=%q", text)
	}
}

func TestBuildLink(t *testing.T) {
	link := BuildLink("+54 9 11 2233-4455", "Juan", sampleItems())

	if !strings.HasPrefix(link, "https://wa.me/5491122334455?text=") {
		t.Fatalf("BuildLink() = %q, se esperaba prefijo wa.me con el número normalizado", link)
	}

	encoded := strings.TrimPrefix(link, "https://wa.me/5491122334455?text=")
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		t.Fatalf("el texto del link no es una query válida: %v", err)
	}
	if !strings.Contains(decoded, "Total: $6000.00") {
		t.Errorf("el texto decodificado no contiene el total; texto=%q", decoded)
	}
}

func TestBuildLinkFromJSON(t *testing.T) {
	raw := []byte(`[{"variant_id":"v1","product_name":"Remera","variant_name":"Rojo / S","quantity":2,"unit_price":1500}]`)

	link, err := BuildLinkFromJSON("5491122334455", "Juan", raw)
	if err != nil {
		t.Fatalf("BuildLinkFromJSON() error inesperado: %v", err)
	}
	if !strings.Contains(link, "https://wa.me/5491122334455?text=") {
		t.Errorf("link inesperado: %q", link)
	}
}

func TestBuildLinkFromJSON_Invalido(t *testing.T) {
	if _, err := BuildLinkFromJSON("54911", "Juan", []byte("no-es-json")); err == nil {
		t.Fatal("se esperaba error con JSON inválido")
	}
}
