package inicio

import (
	"strings"
	"testing"

	"tp2-krohn-trapani-cicopiedi/plataforma/pruebas"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

func TestHomePage_SinSesion(t *testing.T) {
	html := pruebas.Renderizar(t, HomePage(vista.Pagina{Titulo: "Inicio", Clave: "inicio", Servicios: pruebas.Catalogo}))

	pruebas.VerificarContiene(t, html,
		"<!doctype html>",
		`href="/static/styles.css"`,
		`href="/login"`,
		`href="/registro"`,
		// panel de servicios agrupado por categoría, con su logo
		"panel-servicios",
		"Streaming", "Música", "Deportes",
		`class="icono-categoria"`,
		"Netflix",
		// TP6: el único script es HTMX, desde el CDN (no hay archivos .js propios)
		`<script src="https://unpkg.com/htmx.org@1.9.10"></script>`,
	)
	if n := strings.Count(html, "<script"); n != 1 {
		t.Errorf("la página tendría que tener un solo <script> (HTMX), tiene %d", n)
	}
	// sin sesión los servicios no son links para suscribirse
	pruebas.VerificarNoContiene(t, html, "/cuenta/suscripciones/nueva?servicio=", "Cerrar sesión")
}

func TestHomePage_ConSesion(t *testing.T) {
	html := pruebas.Renderizar(t, HomePage(vista.Pagina{Clave: "inicio", Sesion: &vista.Sesion{Nombre: "Nico"}, Servicios: pruebas.Catalogo}))

	pruebas.VerificarContiene(t, html,
		"Hola, Nico",
		"Cerrar sesión",
		`href="/cuenta"`,
		`href="/cuenta/suscripciones/nueva?servicio=1"`,
	)
}
