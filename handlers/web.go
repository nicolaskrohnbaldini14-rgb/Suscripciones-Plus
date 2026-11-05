package handlers

import (
	"net/http"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/logic"
)

// Web reúne los handlers que renderizan HTML con templ. Reemplazan a los
// handlers JSON del TP4: la vista se genera en el servidor y no hay JavaScript.
//
// Las rutas están todas en Registrar. Los handlers se separan por entidad
// (usuarios.go y suscripciones.go) y lo que comparten está en comunes.go
// (sesión de la página, renderizado y errores) y formularios.go (lectura de
// formularios y filtros), como sugiere la clase de lógica del negocio:
// handlers separados por rutas, funciones auxiliares y errores centralizados.
type Web struct {
	usuarios      *logic.ServicioUsuario
	servicios     *logic.ServicioServicio
	suscripciones *logic.ServicioSuscripcion
	sesiones      *Sesiones
}

func NuevoWeb(q *sqlc.Queries, sesiones *Sesiones) *Web {
	return &Web{
		usuarios:      logic.NuevoServicioUsuario(q),
		servicios:     logic.NuevoServicioServicio(q),
		suscripciones: logic.NuevoServicioSuscripcion(q),
		sesiones:      sesiones,
	}
}

// manejadorConUsuario es un handler que recibe al usuario que inició sesión
type manejadorConUsuario func(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario)

// Registrar asocia cada ruta con su handler
func (h *Web) Registrar(mux *http.ServeMux) {
	// Páginas públicas
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /login", h.formLogin)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("GET /registro", h.formRegistro)
	mux.HandleFunc("POST /registro", h.registrar)
	mux.HandleFunc("POST /logout", h.logout)

	// Con sesión iniciada
	mux.HandleFunc("GET /cuenta", h.conSesion(h.cuenta))
	mux.HandleFunc("GET /cuenta/suscripciones", h.conSesion(h.misSuscripciones))
	mux.HandleFunc("GET /cuenta/suscripciones/nueva", h.conSesion(h.nuevaSuscripcion))
	mux.HandleFunc("POST /cuenta/suscripciones/nueva/catalogo", h.conSesion(h.elegirServicio))
	mux.HandleFunc("POST /cuenta/suscripciones", h.conSesion(h.crearSuscripcion))
	mux.HandleFunc("POST /cuenta/suscripciones/{id}/modificar", h.conSesion(h.modificar))
	mux.HandleFunc("POST /cuenta/suscripciones/{id}/password", h.conSesion(h.revelarPassword))
	mux.HandleFunc("POST /cuenta/suscripciones/{id}/eliminar", h.conSesion(h.eliminar))

	// "no-cache": el navegador siempre consulta si hay una versión nueva del CSS
	estaticos := http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))
	mux.Handle("GET /static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		estaticos.ServeHTTP(w, r)
	}))
}
