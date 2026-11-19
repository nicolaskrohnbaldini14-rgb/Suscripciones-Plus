// Package missuscripciones es el slice de "Mis suscripciones": la lista del
// mes con el buscador y el calendario, el detalle, modificar, ver la
// contraseña de la cuenta y eliminar (con HTMX).
package missuscripciones

import (
	"net/http"
	"sort"
	"strings"
	"time"

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
	mux.HandleFunc("GET /cuenta/suscripciones", app.ConSesion(h.misSuscripciones))
	mux.HandleFunc("POST /cuenta/suscripciones/{id}/modificar", app.ConSesion(h.modificar))
	mux.HandleFunc("POST /cuenta/suscripciones/{id}/password", app.ConSesion(h.revelarPassword))
	// TP6: borrado con HTMX (hx-delete), sin recargar la página
	mux.HandleFunc("GET /cuenta/suscripciones/{id}/confirmar-eliminar", app.ConSesion(h.confirmarEliminar))
	mux.HandleFunc("DELETE /cuenta/suscripciones/{id}", app.ConSesion(h.eliminar))
}

// opcionesLista indica qué mostrar en "Mis suscripciones"
type opcionesLista struct {
	Mes              string
	Busqueda         string
	Ver              int32
	Editar           bool
	Aviso            vista.Aviso
	FormEditar       *FormModificar
	ErrorEditar      string
	PasswordRevelada bool
	Password         *string
	ErrorPassword    string
}

func (h *handler) misSuscripciones(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	q := r.URL.Query()
	h.renderSuscripciones(w, r, usr, http.StatusOK, opcionesLista{
		Mes:      plataforma.MesValido(q.Get("mes")),
		Busqueda: strings.TrimSpace(q.Get("q")),
		Ver:      plataforma.LeerEntero(q.Get("ver")),
		Editar:   q.Get("editar") == "1",
		Aviso:    plataforma.AvisoDeLaURL(r),
	})
}

// renderSuscripciones arma la página con las suscripciones que vencen en el
// mes elegido, el buscador, el calendario y, si se pidió, el detalle
func (h *handler) renderSuscripciones(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, estado int, op opcionesLista) {
	// ListarPorUsuario también renueva las suscripciones que vencieron
	todas, err := h.app.Suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return
	}
	d := datosDelMes(todas, op)

	// detalle de una suscripción del usuario (puede ser de cualquier mes)
	if op.Ver > 0 {
		for _, s := range todas {
			if s.IDSuscripcion != op.Ver {
				continue
			}
			det := &Detalle{
				Sub:              s,
				Editable:         d.Editables[s.IDSuscripcion],
				PasswordRevelada: op.PasswordRevelada,
				Password:         op.Password,
				ErrorPassword:    op.ErrorPassword,
				ErrorEditar:      op.ErrorEditar,
			}
			det.Editando = det.Editable && (op.Editar || op.FormEditar != nil)
			if op.FormEditar != nil {
				det.Form = *op.FormEditar
			} else {
				det.Form = FormModificar{Monto: s.Monto, UsuarioCuenta: plataforma.Valor(s.UsuarioCuenta), Estado: s.Estado}
			}
			d.Detalle = det
		}
	}

	p := h.app.Pagina(r, "Mis suscripciones", "suscripciones", &usr)
	p.Aviso = op.Aviso
	plataforma.Render(w, r, estado, MisSuscripcionesPage(p, d))
}

// datosDelMes arma la lista, el gasto y el calendario del mes elegido (sin
// el detalle). También la usa el DELETE para actualizar el gasto y el
// calendario después de borrar.
func datosDelMes(todas []vista.Fila, op opcionesLista) DatosSuscripciones {
	delMes := plataforma.FiltrarPorMes(todas, op.Mes)
	sort.SliceStable(delMes, func(i, j int) bool {
		return delMes[i].FechaVencimiento.Before(delMes[j].FechaVencimiento)
	})

	mesActual := plataforma.MesActual()
	d := DatosSuscripciones{
		Mes:           op.Mes,
		NombreMes:     vista.NombreMes(op.Mes),
		EsMesActual:   op.Mes == mesActual,
		SoloLectura:   op.Mes < mesActual,
		MesAnterior:   vista.SumarMeses(op.Mes, -1),
		MesSiguiente:  vista.SumarMeses(op.Mes, 1),
		Busqueda:      op.Busqueda,
		Suscripciones: buscar(delMes, op.Busqueda),
		HayEnElMes:    len(delMes) > 0,
		HayAlguna:     len(todas) > 0,
		Calendario:    ArmarCalendario(op.Mes, delMes, time.Now()),
		Editables:     map[int32]bool{},
	}
	// mes anterior: todo lo que venció; mes actual o siguientes: lo que falta pagar
	if d.SoloLectura {
		d.Total = vista.TotalActivo(delMes)
	} else {
		d.Total = GastoPendiente(delMes, time.Now())
	}
	d.NombreAnterior = vista.NombreMes(d.MesAnterior)
	d.NombreSiguiente = vista.NombreMes(d.MesSiguiente)
	for _, s := range todas {
		d.Editables[s.IDSuscripcion] = EsEditable(s)
	}
	return d
}

// suscripcionDelUsuario lee el id de la ruta y verifica que la suscripción
// sea del usuario; si no, responde 404 (como si no existiera)
func (h *handler) suscripcionDelUsuario(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, op opcionesLista) (sqlc.Suscripcion, bool) {
	id := plataforma.LeerEntero(r.PathValue("id"))
	propia, err := h.app.Suscripciones.PerteneceA(r.Context(), id, usr.IDUsuario)
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return sqlc.Suscripcion{}, false
	}
	if !propia {
		op.Aviso = plataforma.AvisoError("Suscripción no encontrada.")
		h.renderSuscripciones(w, r, usr, http.StatusNotFound, op)
		return sqlc.Suscripcion{}, false
	}
	sub, err := h.app.Suscripciones.ObtenerPorID(r.Context(), id)
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return sqlc.Suscripcion{}, false
	}
	return sub, true
}

// opcionesDelFormulario conserva el mes y la búsqueda de la lista
func opcionesDelFormulario(r *http.Request) opcionesLista {
	return opcionesLista{
		Mes:      plataforma.MesValido(r.FormValue("mes")),
		Busqueda: strings.TrimSpace(r.FormValue("q")),
	}
}
