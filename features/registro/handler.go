// Package registro es el slice de crear una cuenta de usuario.
package registro

import (
	"net/http"
	"strings"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/plataforma"
)

type handler struct {
	app *plataforma.App
}

// Registrar asocia las rutas del slice con sus handlers
func Registrar(mux *http.ServeMux, app *plataforma.App) {
	h := &handler{app: app}
	mux.HandleFunc("GET /registro", h.formRegistro)
	mux.HandleFunc("POST /registro", h.registrar)
}

func (h *handler) formRegistro(w http.ResponseWriter, r *http.Request) {
	if h.app.UsuarioActual(r) != nil {
		http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
		return
	}
	plataforma.Render(w, r, http.StatusOK, RegistroPage(h.app.Pagina(r, "Registrarse", "registro", nil), DatosRegistro{}))
}

// registrar crea la cuenta y deja la sesión iniciada
func (h *handler) registrar(w http.ResponseWriter, r *http.Request) {
	if !plataforma.ParsearFormulario(w, r) {
		return
	}
	datos := DatosRegistro{
		Nombre: strings.TrimSpace(r.FormValue("nombre")),
		Email:  strings.TrimSpace(r.FormValue("email")),
	}

	usr, err := h.app.Usuarios.Crear(r.Context(), sqlc.CreateUsuarioParams{
		Nombre:   datos.Nombre,
		Email:    datos.Email,
		Password: r.FormValue("password"),
	})
	if err != nil {
		estado, mensaje := plataforma.DescribirError(err)
		p := h.app.Pagina(r, "Registrarse", "registro", nil)
		p.Aviso = plataforma.AvisoError(mensaje)
		plataforma.Render(w, r, estado, RegistroPage(p, datos))
		return
	}

	h.app.IniciarSesionYRedirigir(w, r, usr, "/cuenta?ok=bienvenida")
}
