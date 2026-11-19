// Package inicio es el slice de la portada pública.
package inicio

import (
	"net/http"

	"tp2-krohn-trapani-cicopiedi/plataforma"
)

type handler struct {
	app *plataforma.App
}

// Registrar asocia las rutas del slice con sus handlers
func Registrar(mux *http.ServeMux, app *plataforma.App) {
	h := &handler{app: app}
	mux.HandleFunc("GET /{$}", h.inicio)
}

func (h *handler) inicio(w http.ResponseWriter, r *http.Request) {
	p := h.app.Pagina(r, "Inicio", "inicio", h.app.UsuarioActual(r))
	plataforma.Render(w, r, http.StatusOK, HomePage(p))
}
