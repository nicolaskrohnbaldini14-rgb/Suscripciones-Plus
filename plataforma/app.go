package plataforma

import (
	"log"
	"net/http"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/logic"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

// App reúne lo que comparten todos los slices de features/: los servicios de
// la capa de lógica (que usan las consultas de sqlc) y las sesiones. Es la
// "plataforma" de la clase de diseño (Vertical Slice Architecture): el código
// transversal, como la autenticación y la configuración de la base, que no
// pertenece a ninguna funcionalidad en particular.
type App struct {
	Usuarios      *logic.ServicioUsuario
	Servicios     *logic.ServicioServicio
	Suscripciones *logic.ServicioSuscripcion
	Sesiones      *Sesiones
}

func NuevaApp(q *sqlc.Queries) *App {
	return &App{
		Usuarios:      logic.NuevoServicioUsuario(q),
		Servicios:     logic.NuevoServicioServicio(q),
		Suscripciones: logic.NuevoServicioSuscripcion(q),
		Sesiones:      NuevasSesiones(),
	}
}

// ManejadorConUsuario es un handler que recibe al usuario que inició sesión
type ManejadorConUsuario func(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario)

// UsuarioActual devuelve el usuario de la sesión, o nil si no ingresó
func (a *App) UsuarioActual(r *http.Request) *sqlc.Usuario {
	id, ok := a.Sesiones.Usuario(r)
	if !ok {
		return nil
	}
	usr, err := a.Usuarios.ObtenerPorID(r.Context(), id)
	if err != nil {
		return nil
	}
	return &usr
}

// ConSesion exige una sesión iniciada; si no la hay, redirige al login
func (a *App) ConSesion(siguiente ManejadorConUsuario) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usr := a.UsuarioActual(r)
		if usr == nil {
			// en un pedido de HTMX una redirección común se seguiría por AJAX y
			// la página de login quedaría dentro del fragmento: con HX-Redirect
			// HTMX lleva al navegador entero al login
			if EsHTMX(r) {
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

// Pagina arma los datos que usa el layout: sesión y catálogo (para el panel
// de servicios de la barra)
func (a *App) Pagina(r *http.Request, titulo, clave string, usr *sqlc.Usuario) vista.Pagina {
	p := vista.Pagina{Titulo: titulo, Clave: clave}
	if usr != nil {
		p.Sesion = &vista.Sesion{Nombre: usr.Nombre}
	}
	if servicios, err := a.Servicios.Listar(r.Context()); err == nil {
		p.Servicios = servicios
	} else {
		log.Printf("Error al listar servicios: %v", err)
	}
	return p
}

// IniciarSesionYRedirigir deja la sesión iniciada (login y registro)
func (a *App) IniciarSesionYRedirigir(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, destino string) {
	if err := a.Sesiones.Iniciar(w, usr.IDUsuario); err != nil {
		ErrorInterno(w, err)
		return
	}
	http.Redirect(w, r, destino, http.StatusSeeOther)
}
