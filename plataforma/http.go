package plataforma

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"

	"tp2-krohn-trapani-cicopiedi/logic"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

// Lo que comparten todos los handlers: los avisos, el renderizado, HTMX y el
// manejo centralizado de errores.

// AvisosOK son los avisos de éxito después de redirigir (patrón
// Post/Redirect/Get) o en las respuestas de HTMX
var AvisosOK = map[string]string{
	"bienvenida": "¡Listo! Tu cuenta se creó y ya iniciaste sesión.",
	"creada":     "Suscripción creada.",
	"modificada": "Suscripción actualizada.",
	"eliminada":  "Suscripción eliminada.",
}

func AvisoDeLaURL(r *http.Request) vista.Aviso {
	return vista.Aviso{Texto: AvisosOK[r.URL.Query().Get("ok")]}
}

func AvisoError(mensaje string) vista.Aviso {
	return vista.Aviso{Texto: mensaje, Error: true}
}

// EsHTMX indica si el pedido lo hizo HTMX (envía el encabezado HX-Request)
func EsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// CambiarDestinoHTMX hace que HTMX ponga la respuesta en otro lugar que el
// hx-target del elemento (encabezados HX-Retarget y HX-Reswap). Se usa para
// los errores: HTMX 1.9 no reemplaza nada si el código es 4xx o 5xx, así que
// a HTMX se le responde 200 con el error en el fragmento. Sin HTMX (por
// ejemplo desde Hurl) se conserva el código de error. Devuelve el código.
func CambiarDestinoHTMX(w http.ResponseWriter, r *http.Request, estado int, destino, swap string) int {
	if !EsHTMX(r) {
		return estado
	}
	w.Header().Set("HX-Retarget", destino)
	w.Header().Set("HX-Reswap", swap)
	return http.StatusOK
}

func Render(w http.ResponseWriter, r *http.Request, estado int, componente templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(estado)
	if err := componente.Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar la página: %v", err)
	}
}

func ErrorInterno(w http.ResponseWriter, err error) {
	log.Printf("Error interno: %v", err)
	http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
}

// DescribirError traduce los errores de la capa de lógica a un código HTTP y
// al mensaje que se le muestra al usuario. Los errores inesperados no se
// muestran tal cual para no filtrar detalles internos.
func DescribirError(err error) (int, string) {
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

// Estaticos sirve /static/ (el CSS). "no-cache": el navegador siempre
// consulta si hay una versión nueva
func Estaticos() http.Handler {
	estaticos := http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		estaticos.ServeHTTP(w, r)
	})
}
