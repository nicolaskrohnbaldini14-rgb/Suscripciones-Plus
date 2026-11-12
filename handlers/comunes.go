package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/logic"
	"tp2-krohn-trapani-cicopiedi/views"
)

// Lo que comparten todas las páginas: el usuario de la sesión, los datos del
// layout, los avisos, el renderizado y el manejo centralizado de errores.

// usuarioActual devuelve el usuario de la sesión, o nil si no ingresó
func (h *Web) usuarioActual(r *http.Request) *sqlc.Usuario {
	id, ok := h.sesiones.Usuario(r)
	if !ok {
		return nil
	}
	usr, err := h.usuarios.ObtenerPorID(r.Context(), id)
	if err != nil {
		return nil
	}
	return &usr
}

// conSesion exige una sesión iniciada; si no la hay, redirige al login
func (h *Web) conSesion(siguiente manejadorConUsuario) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usr := h.usuarioActual(r)
		if usr == nil {
			// en un pedido de HTMX una redirección común se seguiría por AJAX y
			// la página de login quedaría dentro del fragmento: con HX-Redirect
			// HTMX lleva al navegador entero al login
			if esHTMX(r) {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		siguiente(w, r, *usr)
	}
}

// pagina arma los datos que usa el layout: sesión y catálogo (para el panel
// de servicios de la barra)
func (h *Web) pagina(r *http.Request, titulo, clave string, usr *sqlc.Usuario) views.Pagina {
	p := views.Pagina{Titulo: titulo, Clave: clave}
	if usr != nil {
		p.Sesion = &views.Sesion{Nombre: usr.Nombre}
	}
	if servicios, err := h.servicios.Listar(r.Context()); err == nil {
		p.Servicios = servicios
	} else {
		log.Printf("Error al listar servicios: %v", err)
	}
	return p
}

// avisos de éxito después de redirigir (patrón Post/Redirect/Get)
var avisosOK = map[string]string{
	"bienvenida": "¡Listo! Tu cuenta se creó y ya iniciaste sesión.",
	"creada":     "Suscripción creada.",
	"modificada": "Suscripción actualizada.",
	"eliminada":  "Suscripción eliminada.",
}

func avisoDeLaURL(r *http.Request) views.Aviso {
	return views.Aviso{Texto: avisosOK[r.URL.Query().Get("ok")]}
}

func avisoError(mensaje string) views.Aviso {
	return views.Aviso{Texto: mensaje, Error: true}
}

// esHTMX indica si el pedido lo hizo HTMX (envía el encabezado HX-Request)
func esHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// cambiarDestinoHTMX hace que HTMX ponga la respuesta en otro lugar que el
// hx-target del elemento (encabezados HX-Retarget y HX-Reswap). Se usa para
// los errores: HTMX 1.9 no reemplaza nada si el código es 4xx o 5xx, así que
// a HTMX se le responde 200 con el error en el fragmento. Sin HTMX (por
// ejemplo desde Hurl) se conserva el código de error. Devuelve el código.
func cambiarDestinoHTMX(w http.ResponseWriter, r *http.Request, estado int, destino, swap string) int {
	if !esHTMX(r) {
		return estado
	}
	w.Header().Set("HX-Retarget", destino)
	w.Header().Set("HX-Reswap", swap)
	return http.StatusOK
}

func render(w http.ResponseWriter, r *http.Request, estado int, componente templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(estado)
	if err := componente.Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar la página: %v", err)
	}
}

func errorInterno(w http.ResponseWriter, err error) {
	log.Printf("Error interno: %v", err)
	http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
}

// describirError traduce los errores de la capa de lógica a un código HTTP y
// al mensaje que se le muestra al usuario. Los errores inesperados no se
// muestran tal cual para no filtrar detalles internos.
func describirError(err error) (int, string) {
	switch {
	case errors.Is(err, logic.ErrCredenciales):
		return http.StatusUnauthorized, err.Error()
	case errors.Is(err, logic.ErrSoloLectura), errors.Is(err, logic.ErrPasswordUsuario):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, logic.ErrCuentaRequerida):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, logic.ErrCuentaRepetida):
		return http.StatusConflict, err.Error()
	}

	msg := err.Error()
	contiene := func(palabras ...string) bool {
		for _, p := range palabras {
			if strings.Contains(msg, p) {
				return true
			}
		}
		return false
	}

	switch {
	case contiene("no encontrado", "no encontrada", "no existe"):
		return http.StatusNotFound, msg
	case contiene("ya está registrado", "en uso"):
		return http.StatusConflict, msg
	case contiene("obligatori", "válid", "mayor a cero", "anterior", "posterior", "siempre en estado"):
		return http.StatusBadRequest, msg
	default:
		log.Printf("Error interno: %v", err)
		return http.StatusInternalServerError, "Error interno del servidor"
	}
}
