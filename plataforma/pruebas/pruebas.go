// Package pruebas tiene las funciones que usan los tests de las vistas de
// cada slice (renderizar un componente y buscar texto en el HTML) y los datos
// de prueba que comparten. Solo lo importan los archivos _test.go.
package pruebas

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

func Renderizar(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("Error al renderizar el componente: %v", err)
	}
	return buf.String()
}

func VerificarContiene(t *testing.T, html string, esperados ...string) {
	t.Helper()
	for _, e := range esperados {
		if !strings.Contains(html, e) {
			t.Errorf("La página debería contener %q", e)
		}
	}
}

func VerificarNoContiene(t *testing.T, html string, prohibidos ...string) {
	t.Helper()
	for _, p := range prohibidos {
		if strings.Contains(html, p) {
			t.Errorf("La página no debería contener %q", p)
		}
	}
}

func Fecha(anio int, mes time.Month, dia int) time.Time {
	return time.Date(anio, mes, dia, 0, 0, 0, 0, time.UTC)
}

// Catalogo es un catálogo chico de servicios para los tests
var Catalogo = []sqlc.Servicio{
	{IDServicio: 1, Nombre: "Netflix", Categoria: "Streaming"},
	{IDServicio: 2, Nombre: "Spotify", Categoria: "Música"},
	{IDServicio: 3, Nombre: "Pádel", Categoria: "Deportes"},
}

// Fila es una suscripción activa de ejemplo que vence el 28/10/2026
func Fila() vista.Fila {
	cuenta := "martinagym"
	return vista.Fila{
		IDSuscripcion:     5,
		Monto:             "32000.00",
		FechaInicio:       Fecha(2026, 10, 1),
		FechaVencimiento:  Fecha(2026, 10, 28),
		UsuarioCuenta:     &cuenta,
		Estado:            "Activa",
		ServicioNombre:    "Gimnasio",
		ServicioCategoria: "Deportes",
	}
}
