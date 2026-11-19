// Package sesion es el slice de iniciar y cerrar sesión.
package sesion

import (
	"errors"
	"net/http"
	"strings"

	"tp2-krohn-trapani-cicopiedi/logic"
	"tp2-krohn-trapani-cicopiedi/plataforma"
)

type handler struct {
	app *plataforma.App
}

// Registrar asocia las rutas del slice con sus handlers
func Registrar(mux *http.ServeMux, app *plataforma.App) {
	h := &handler{app: app}
	mux.HandleFunc("GET /login", h.formLogin)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /logout", h.logout)
}

func (h *handler) formLogin(w http.ResponseWriter, r *http.Request) {
	if h.app.UsuarioActual(r) != nil {
		http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
		return
	}
	plataforma.Render(w, r, http.StatusOK, LoginPage(h.app.Pagina(r, "Iniciar sesión", "login", nil), ""))
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	if !plataforma.ParsearFormulario(w, r) {
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))

	usr, err := h.app.Usuarios.Autenticar(r.Context(), email, r.FormValue("password"))
	if err != nil {
		if errors.Is(err, logic.ErrCredenciales) {
			plataforma.IniciosDeSesion.WithLabelValues("error").Inc()
		}
		estado, mensaje := plataforma.DescribirError(err)
		p := h.app.Pagina(r, "Iniciar sesión", "login", nil)
		p.Aviso = plataforma.AvisoError(mensaje)
		plataforma.Render(w, r, estado, LoginPage(p, email))
		return
	}

	plataforma.IniciosDeSesion.WithLabelValues("ok").Inc()
	h.app.IniciarSesionYRedirigir(w, r, usr, "/cuenta")
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	h.app.Sesiones.Cerrar(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
