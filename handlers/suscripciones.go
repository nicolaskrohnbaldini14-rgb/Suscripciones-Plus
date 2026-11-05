package handlers

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/views"
)

// Handlers de las suscripciones: lista del mes, alta, modificar, ver la
// contraseña de la cuenta y eliminar.

// opcionesLista indica qué mostrar en "Mis suscripciones"
type opcionesLista struct {
	Mes              string
	Busqueda         string
	Ver              int32
	Editar           bool
	Aviso            views.Aviso
	FormEditar       *views.FormModificar
	ErrorEditar      string
	PasswordRevelada bool
	Password         *string
	ErrorPassword    string
}

func (h *Web) misSuscripciones(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	q := r.URL.Query()
	h.renderSuscripciones(w, r, usr, http.StatusOK, opcionesLista{
		Mes:      mesValido(q.Get("mes")),
		Busqueda: strings.TrimSpace(q.Get("q")),
		Ver:      leerEntero(q.Get("ver")),
		Editar:   q.Get("editar") == "1",
		Aviso:    avisoDeLaURL(r),
	})
}

// renderSuscripciones arma la página con las suscripciones que vencen en el
// mes elegido, el buscador, el calendario y, si se pidió, el detalle
func (h *Web) renderSuscripciones(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, estado int, op opcionesLista) {
	// ListarPorUsuario también renueva las suscripciones que vencieron
	todas, err := h.suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		errorInterno(w, err)
		return
	}

	delMes := filtrarPorMes(todas, op.Mes)
	sort.SliceStable(delMes, func(i, j int) bool {
		return delMes[i].FechaVencimiento.Before(delMes[j].FechaVencimiento)
	})

	d := views.DatosSuscripciones{
		Mes:           op.Mes,
		NombreMes:     views.NombreMes(op.Mes),
		EsMesActual:   op.Mes == mesActual(),
		SoloLectura:   op.Mes < mesActual(),
		MesAnterior:   views.SumarMeses(op.Mes, -1),
		MesSiguiente:  views.SumarMeses(op.Mes, 1),
		Busqueda:      op.Busqueda,
		Suscripciones: buscar(delMes, op.Busqueda),
		HayEnElMes:    len(delMes) > 0,
		HayAlguna:     len(todas) > 0,
		Calendario:    views.ArmarCalendario(op.Mes, delMes, time.Now()),
		Editables:     map[int32]bool{},
	}
	// mes anterior: todo lo que venció; mes actual o siguientes: lo que falta pagar
	if d.SoloLectura {
		d.Total = views.TotalActivo(delMes)
	} else {
		d.Total = views.GastoPendiente(delMes, time.Now())
	}
	d.NombreAnterior = views.NombreMes(d.MesAnterior)
	d.NombreSiguiente = views.NombreMes(d.MesSiguiente)
	for _, s := range todas {
		d.Editables[s.IDSuscripcion] = views.EsEditable(s)
	}

	// detalle de una suscripción del usuario (puede ser de cualquier mes)
	if op.Ver > 0 {
		for _, s := range todas {
			if s.IDSuscripcion != op.Ver {
				continue
			}
			det := &views.Detalle{
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
				det.Form = views.FormModificar{Monto: s.Monto, UsuarioCuenta: valor(s.UsuarioCuenta), Estado: s.Estado}
			}
			d.Detalle = det
		}
	}

	p := h.pagina(r, "Mis suscripciones", "suscripciones", &usr)
	p.Aviso = op.Aviso
	render(w, r, estado, views.MisSuscripcionesPage(p, d))
}

func (h *Web) nuevaSuscripcion(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	q := r.URL.Query()
	form := views.FormSuscripcion{IDServicio: leerEntero(q.Get("servicio"))}

	p := h.pagina(r, "Nueva suscripción", "nueva-suscripcion", &usr)
	// ?catalogo=1 abre el panel de servicios en modo "elegir"
	if q.Get("catalogo") == "1" {
		p.Catalogo = views.Catalogo{
			Abierto:   true,
			Elegir:    true,
			IDActual:  form.IDServicio,
			CerrarURL: "/cuenta/suscripciones/nueva?servicio=" + strconv.Itoa(int(form.IDServicio)),
		}
	}
	render(w, r, http.StatusOK, views.NuevaSuscripcionPage(p, form))
}

// crearSuscripcion guarda la suscripción a nombre del usuario de la sesión
// (siempre Activa) y redirige al mes en que vence (Post/Redirect/Get)
func (h *Web) crearSuscripcion(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !parsearFormulario(w, r) {
		return
	}
	form := views.FormSuscripcion{
		IDServicio:       leerEntero(r.FormValue("id_servicio")),
		Monto:            strings.TrimSpace(r.FormValue("monto")),
		FechaInicio:      r.FormValue("fecha_inicio"),
		FechaVencimiento: r.FormValue("fecha_vencimiento"),
		UsuarioCuenta:    strings.TrimSpace(r.FormValue("usuario_cuenta")),
	}

	// los valores inválidos quedan en cero y los rechaza la capa de lógica;
	// usuario y contraseña vacíos quedan en NULL
	nueva, err := h.suscripciones.Crear(r.Context(), sqlc.CreateSubscripcionParams{
		IDUsuario:        usr.IDUsuario,
		IDServicio:       form.IDServicio,
		Monto:            form.Monto,
		FechaInicio:      leerFecha(form.FechaInicio),
		FechaVencimiento: leerFecha(form.FechaVencimiento),
		UsuarioCuenta:    texto(form.UsuarioCuenta),
		PasswordCuenta:   texto(r.FormValue("password_cuenta")),
	})
	if err != nil {
		estado, mensaje := describirError(err)
		p := h.pagina(r, "Nueva suscripción", "nueva-suscripcion", &usr)
		p.Aviso = avisoError(mensaje)
		render(w, r, estado, views.NuevaSuscripcionPage(p, form))
		return
	}

	destino := "/cuenta/suscripciones?" + url.Values{
		"mes": {nueva.FechaVencimiento.Format("2006-01")},
		"ok":  {"creada"},
	}.Encode()
	http.Redirect(w, r, destino, http.StatusSeeOther)
}

// suscripcionDelUsuario lee el id de la ruta y verifica que la suscripción
// sea del usuario; si no, responde 404 (como si no existiera)
func (h *Web) suscripcionDelUsuario(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, op opcionesLista) (sqlc.Suscripcion, bool) {
	id := leerEntero(r.PathValue("id"))
	propia, err := h.suscripciones.PerteneceA(r.Context(), id, usr.IDUsuario)
	if err != nil {
		errorInterno(w, err)
		return sqlc.Suscripcion{}, false
	}
	if !propia {
		op.Aviso = avisoError("Suscripción no encontrada.")
		h.renderSuscripciones(w, r, usr, http.StatusNotFound, op)
		return sqlc.Suscripcion{}, false
	}
	sub, err := h.suscripciones.ObtenerPorID(r.Context(), id)
	if err != nil {
		errorInterno(w, err)
		return sqlc.Suscripcion{}, false
	}
	return sub, true
}

// opcionesDelFormulario conserva el mes y la búsqueda de la lista
func opcionesDelFormulario(r *http.Request) opcionesLista {
	return opcionesLista{
		Mes:      mesValido(r.FormValue("mes")),
		Busqueda: strings.TrimSpace(r.FormValue("q")),
	}
}

func redirigirALista(w http.ResponseWriter, r *http.Request, op opcionesLista, ver int32, aviso string) {
	destino := views.UrlSuscripciones(op.Mes, op.Busqueda, ver, false)
	separador := "?"
	if strings.Contains(destino, "?") {
		separador = "&"
	}
	http.Redirect(w, r, destino+separador+"ok="+aviso, http.StatusSeeOther)
}

// modificar cambia el monto, el usuario, (si se escribe) la contraseña de
// la cuenta y el estado (activa o pausada); si falla, vuelve a mostrar el
// detalle con el error
func (h *Web) modificar(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !parsearFormulario(w, r) {
		return
	}
	op := opcionesDelFormulario(r)
	sub, ok := h.suscripcionDelUsuario(w, r, usr, op)
	if !ok {
		return
	}

	form := views.FormModificar{
		Monto:         strings.TrimSpace(r.FormValue("monto")),
		UsuarioCuenta: strings.TrimSpace(r.FormValue("usuario_cuenta")),
		Estado:        r.FormValue("estado"),
	}
	// si el formulario no trae el estado, se conserva el que tenía
	if form.Estado == "" {
		form.Estado = sub.Estado
	}
	_, err := h.suscripciones.Actualizar(r.Context(), sub.IDSuscripcion, sqlc.UpdateSubscripcionParams{
		Monto:            form.Monto,
		FechaVencimiento: sub.FechaVencimiento,
		Estado:           form.Estado,
		UsuarioCuenta:    texto(form.UsuarioCuenta),
		PasswordCuenta:   texto(r.FormValue("password_cuenta")),
	})
	if err != nil {
		estado, mensaje := describirError(err)
		op.Ver = sub.IDSuscripcion
		op.FormEditar = &form
		op.ErrorEditar = mensaje
		h.renderSuscripciones(w, r, usr, estado, op)
		return
	}
	redirigirALista(w, r, op, sub.IDSuscripcion, "modificada")
}

// revelarPassword muestra la contraseña de la cuenta en el detalle, solo si
// el usuario confirma su propia contraseña (no se redirige: no queda en la URL)
func (h *Web) revelarPassword(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !parsearFormulario(w, r) {
		return
	}
	op := opcionesDelFormulario(r)
	sub, ok := h.suscripcionDelUsuario(w, r, usr, op)
	if !ok {
		return
	}
	op.Ver = sub.IDSuscripcion

	password, err := h.suscripciones.RevelarPasswordCuenta(r.Context(), sub.IDSuscripcion, usr.IDUsuario, r.FormValue("password"))
	if err != nil {
		estado, mensaje := describirError(err)
		op.ErrorPassword = mensaje
		h.renderSuscripciones(w, r, usr, estado, op)
		return
	}
	op.PasswordRevelada = true
	op.Password = password
	h.renderSuscripciones(w, r, usr, http.StatusOK, op)
}

func (h *Web) eliminar(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !parsearFormulario(w, r) {
		return
	}
	op := opcionesDelFormulario(r)
	sub, ok := h.suscripcionDelUsuario(w, r, usr, op)
	if !ok {
		return
	}
	if err := h.suscripciones.Eliminar(r.Context(), sub.IDSuscripcion); err != nil {
		estado, mensaje := describirError(err)
		op.Aviso = avisoError(mensaje)
		h.renderSuscripciones(w, r, usr, estado, op)
		return
	}
	redirigirALista(w, r, op, 0, "eliminada")
}
