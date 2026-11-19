// Package nuevasuscripcion es el slice de "Nueva suscripción": el
// formulario, el catálogo para elegir el servicio y el alta con HTMX.
package nuevasuscripcion

import (
	"net/http"
	"strconv"
	"strings"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/plataforma"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

type handler struct {
	app *plataforma.App
}

// Registrar asocia las rutas del slice con sus handlers
func Registrar(mux *http.ServeMux, app *plataforma.App) {
	h := &handler{app: app}
	mux.HandleFunc("GET /cuenta/suscripciones/nueva", app.ConSesion(h.nuevaSuscripcion))
	mux.HandleFunc("POST /cuenta/suscripciones/nueva/catalogo", app.ConSesion(h.elegirServicio))
	mux.HandleFunc("POST /cuenta/suscripciones", app.ConSesion(h.crearSuscripcion))
}

// renderNueva dibuja la página "Nueva suscripción" completa (formulario y
// suscripciones activas). Si el pedido es de HTMX (el botón del catálogo
// está dentro del formulario, que se envía por AJAX), la página reemplaza al
// <body> entero: es lo mismo que haría el navegador sin HTMX.
func (h *handler) renderNueva(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, estado int, p vista.Pagina, form FormSuscripcion) {
	todas, err := h.app.Suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return
	}
	estado = plataforma.CambiarDestinoHTMX(w, r, estado, "body", "innerHTML")
	plataforma.Render(w, r, estado, NuevaSuscripcionPage(p, form, vista.ActivasDe(todas)))
}

// leerFormSuscripcion lee el formulario de nueva suscripción (la contraseña
// de la cuenta no se guarda en el formulario: no se vuelve a mostrar)
func leerFormSuscripcion(r *http.Request) FormSuscripcion {
	return FormSuscripcion{
		IDServicio:       plataforma.LeerEntero(r.FormValue("id_servicio")),
		Monto:            strings.TrimSpace(r.FormValue("monto")),
		FechaInicio:      r.FormValue("fecha_inicio"),
		FechaVencimiento: r.FormValue("fecha_vencimiento"),
		UsuarioCuenta:    strings.TrimSpace(r.FormValue("usuario_cuenta")),
	}
}

func (h *handler) nuevaSuscripcion(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	q := r.URL.Query()
	form := FormSuscripcion{IDServicio: plataforma.LeerEntero(q.Get("servicio"))}

	p := h.app.Pagina(r, "Nueva suscripción", "nueva-suscripcion", &usr)
	// ?catalogo=1 abre el panel de servicios en modo "elegir"
	if q.Get("catalogo") == "1" {
		p.Catalogo = vista.Catalogo{
			Abierto:   true,
			Elegir:    true,
			IDActual:  form.IDServicio,
			CerrarURL: "/cuenta/suscripciones/nueva?servicio=" + strconv.Itoa(int(form.IDServicio)),
		}
	}
	h.renderNueva(w, r, usr, http.StatusOK, p, form)
}

// elegirServicio abre o cierra el catálogo grande para elegir el servicio de
// una nueva suscripción, sin perder lo que ya se escribió en el formulario.
// Los datos llegan por POST (no quedan en la URL) y la página se vuelve a
// dibujar con ellos; la contraseña de la cuenta no se vuelve a mostrar.
//   - Desde el botón "Servicio" del formulario: abre el catálogo.
//   - Desde un servicio del catálogo (o la ×): lo elige y cierra el catálogo.
func (h *handler) elegirServicio(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !plataforma.ParsearFormulario(w, r) {
		return
	}
	form := leerFormSuscripcion(r)

	p := h.app.Pagina(r, "Nueva suscripción", "nueva-suscripcion", &usr)
	if elegido := r.FormValue("elegir_servicio"); elegido != "" {
		form.IDServicio = plataforma.LeerEntero(elegido)
	} else {
		p.Catalogo = vista.Catalogo{
			Abierto:    true,
			Elegir:     true,
			IDActual:   form.IDServicio,
			Formulario: IDFormularioSuscripcion,
		}
	}
	h.renderNueva(w, r, usr, http.StatusOK, p, form)
}

// crearSuscripcion guarda la suscripción a nombre del usuario de la sesión
// (siempre Activa). TP6: ya no redirige. Devuelve solo la lista de
// suscripciones activas actualizada, que HTMX pone en #panel-activas, y
// fuera de banda el formulario vacío y el aviso "Suscripción creada".
func (h *handler) crearSuscripcion(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !plataforma.ParsearFormulario(w, r) {
		return
	}
	form := leerFormSuscripcion(r)

	// el servicio ya no es un <select> con "required": si no se eligió, se avisa
	if form.IDServicio == 0 {
		h.errorAlCrear(w, r, usr, form, http.StatusBadRequest, "Elegí un servicio del catálogo.")
		return
	}

	// los valores inválidos quedan en cero y los rechaza la capa de lógica;
	// usuario y contraseña vacíos quedan en NULL
	_, err := h.app.Suscripciones.Crear(r.Context(), sqlc.CreateSubscripcionParams{
		IDUsuario:        usr.IDUsuario,
		IDServicio:       form.IDServicio,
		Monto:            form.Monto,
		FechaInicio:      plataforma.LeerFecha(form.FechaInicio),
		FechaVencimiento: plataforma.LeerFecha(form.FechaVencimiento),
		UsuarioCuenta:    plataforma.Texto(form.UsuarioCuenta),
		PasswordCuenta:   plataforma.Texto(r.FormValue("password_cuenta")),
	})
	if err != nil {
		estado, mensaje := plataforma.DescribirError(err)
		h.errorAlCrear(w, r, usr, form, estado, mensaje)
		return
	}
	plataforma.SuscripcionesCreadas.Inc()

	// se vuelve a consultar la lista completa (ya con la nueva suscripción)
	todas, err := h.app.Suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return
	}
	servicios, err := h.app.Servicios.Listar(r.Context())
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return
	}
	aviso := vista.Aviso{Texto: plataforma.AvisosOK["creada"]}
	plataforma.Render(w, r, http.StatusOK, RespuestaSuscripcionCreada(servicios, vista.ActivasDe(todas), aviso))
}

// errorAlCrear muestra por qué no se pudo crear la suscripción, sin perder
// lo que se escribió. Con HTMX se devuelve solo el formulario (reemplaza al
// que está en la página) y el aviso con el error (fuera de banda). Sin HTMX
// se dibuja la página completa con el código de error, como antes.
func (h *handler) errorAlCrear(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, form FormSuscripcion, estado int, mensaje string) {
	p := h.app.Pagina(r, "Nueva suscripción", "nueva-suscripcion", &usr)
	p.Aviso = plataforma.AvisoError(mensaje)
	if !plataforma.EsHTMX(r) {
		h.renderNueva(w, r, usr, estado, p, form)
		return
	}
	estado = plataforma.CambiarDestinoHTMX(w, r, estado, "#"+IDFormularioSuscripcion, "outerHTML")
	plataforma.Render(w, r, estado, RespuestaErrorSuscripcion(p.Servicios, form, p.Aviso))
}
