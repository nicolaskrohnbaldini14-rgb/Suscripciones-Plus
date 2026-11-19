package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/logic"
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
	d := datosDelMes(todas, op)

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

// datosDelMes arma la lista, el gasto y el calendario del mes elegido (sin
// el detalle). También la usa el DELETE para actualizar el gasto y el
// calendario después de borrar.
func datosDelMes(todas []views.Fila, op opcionesLista) views.DatosSuscripciones {
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
	return d
}

// activasDe devuelve todas las suscripciones activas (de cualquier mes),
// ordenadas por vencimiento: el panel de "Mi cuenta" y de "Nueva suscripción"
func activasDe(todas []views.Fila) views.Activas {
	var activas []views.Fila
	for _, s := range todas {
		if s.Estado == "Activa" {
			activas = append(activas, s)
		}
	}
	sort.SliceStable(activas, func(i, j int) bool {
		return activas[i].FechaVencimiento.Before(activas[j].FechaVencimiento)
	})
	return views.Activas{Lista: activas}
}

// renderNueva dibuja la página "Nueva suscripción" completa (formulario y
// suscripciones activas). Si el pedido es de HTMX (el botón del catálogo
// está dentro del formulario, que se envía por AJAX), la página reemplaza al
// <body> entero: es lo mismo que haría el navegador sin HTMX.
func (h *Web) renderNueva(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, estado int, p views.Pagina, form views.FormSuscripcion) {
	todas, err := h.suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		errorInterno(w, err)
		return
	}
	estado = cambiarDestinoHTMX(w, r, estado, "body", "innerHTML")
	render(w, r, estado, views.NuevaSuscripcionPage(p, form, activasDe(todas)))
}

// leerFormSuscripcion lee el formulario de nueva suscripción (la contraseña
// de la cuenta no se guarda en el formulario: no se vuelve a mostrar)
func leerFormSuscripcion(r *http.Request) views.FormSuscripcion {
	return views.FormSuscripcion{
		IDServicio:       leerEntero(r.FormValue("id_servicio")),
		Monto:            strings.TrimSpace(r.FormValue("monto")),
		FechaInicio:      r.FormValue("fecha_inicio"),
		FechaVencimiento: r.FormValue("fecha_vencimiento"),
		UsuarioCuenta:    strings.TrimSpace(r.FormValue("usuario_cuenta")),
	}
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
	h.renderNueva(w, r, usr, http.StatusOK, p, form)
}

// elegirServicio abre o cierra el catálogo grande para elegir el servicio de
// una nueva suscripción, sin perder lo que ya se escribió en el formulario.
// Los datos llegan por POST (no quedan en la URL) y la página se vuelve a
// dibujar con ellos; la contraseña de la cuenta no se vuelve a mostrar.
//   - Desde el botón "Servicio" del formulario: abre el catálogo.
//   - Desde un servicio del catálogo (o la ×): lo elige y cierra el catálogo.
func (h *Web) elegirServicio(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !parsearFormulario(w, r) {
		return
	}
	form := leerFormSuscripcion(r)

	p := h.pagina(r, "Nueva suscripción", "nueva-suscripcion", &usr)
	if elegido := r.FormValue("elegir_servicio"); elegido != "" {
		form.IDServicio = leerEntero(elegido)
	} else {
		p.Catalogo = views.Catalogo{
			Abierto:    true,
			Elegir:     true,
			IDActual:   form.IDServicio,
			Formulario: views.IDFormularioSuscripcion,
		}
	}
	h.renderNueva(w, r, usr, http.StatusOK, p, form)
}

// crearSuscripcion guarda la suscripción a nombre del usuario de la sesión
// (siempre Activa). TP6: ya no redirige. Devuelve solo la lista de
// suscripciones activas actualizada, que HTMX pone en #panel-activas, y
// fuera de banda el formulario vacío y el aviso "Suscripción creada".
func (h *Web) crearSuscripcion(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !parsearFormulario(w, r) {
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
	_, err := h.suscripciones.Crear(r.Context(), sqlc.CreateSubscripcionParams{
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
		h.errorAlCrear(w, r, usr, form, estado, mensaje)
		return
	}

	// se vuelve a consultar la lista completa (ya con la nueva suscripción)
	todas, err := h.suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		errorInterno(w, err)
		return
	}
	servicios, err := h.servicios.Listar(r.Context())
	if err != nil {
		errorInterno(w, err)
		return
	}
	render(w, r, http.StatusOK, views.RespuestaSuscripcionCreada(servicios, activasDe(todas), views.Aviso{Texto: avisosOK["creada"]}))
}

// errorAlCrear muestra por qué no se pudo crear la suscripción, sin perder
// lo que se escribió. Con HTMX se devuelve solo el formulario (reemplaza al
// que está en la página) y el aviso con el error (fuera de banda). Sin HTMX
// se dibuja la página completa con el código de error, como antes.
func (h *Web) errorAlCrear(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, form views.FormSuscripcion, estado int, mensaje string) {
	p := h.pagina(r, "Nueva suscripción", "nueva-suscripcion", &usr)
	p.Aviso = avisoError(mensaje)
	if !esHTMX(r) {
		h.renderNueva(w, r, usr, estado, p, form)
		return
	}
	estado = cambiarDestinoHTMX(w, r, estado, "#"+views.IDFormularioSuscripcion, "outerHTML")
	render(w, r, estado, views.RespuestaErrorSuscripcion(p.Servicios, form, p.Aviso))
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

// confirmarEliminar maneja GET /cuenta/suscripciones/{id}/confirmar-eliminar
// (TP6): devuelve el aviso que pregunta si se quiere eliminar la suscripción
// (en lugar de la ventana del navegador), o el aviso vacío si se canceló.
func (h *Web) confirmarEliminar(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	q := r.URL.Query()
	if q.Get("cancelar") == "1" {
		render(w, r, http.StatusOK, views.Avisos(views.Aviso{}, false))
		return
	}

	id := leerEntero(r.PathValue("id"))
	todas, err := h.suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		errorInterno(w, err)
		return
	}
	for _, s := range todas {
		if s.IDSuscripcion != id {
			continue
		}
		// las de solo lectura (vencidas o de un mes anterior) no se borran
		if logic.EsSoloLectura(s.Estado, s.FechaVencimiento) {
			estado, mensaje := describirError(logic.ErrSoloLectura)
			errorAlEliminar(w, r, estado, mensaje)
			return
		}
		render(w, r, http.StatusOK, views.ConfirmarEliminar(mesValido(q.Get("mes")), strings.TrimSpace(q.Get("q")), s))
		return
	}
	// no existe o es de otro usuario
	errorAlEliminar(w, r, http.StatusNotFound, "Suscripción no encontrada.")
}

// eliminar maneja DELETE /cuenta/suscripciones/{id} (TP6): borra la
// suscripción y responde 200 con el contenido principal vacío, así HTMX
// reemplaza la fila (hx-target="#fila-{id}") por nada y desaparece de la
// lista. Fuera de banda se actualizan el gasto del mes, el calendario y el
// aviso. Si la lista del mes queda vacía se reemplaza la lista entera, para
// mostrar el mensaje de lista vacía.
func (h *Web) eliminar(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	q := r.URL.Query()
	op := opcionesLista{Mes: mesValido(q.Get("mes")), Busqueda: strings.TrimSpace(q.Get("q"))}

	id := leerEntero(r.PathValue("id"))
	propia, err := h.suscripciones.PerteneceA(r.Context(), id, usr.IDUsuario)
	if err != nil {
		errorInterno(w, err)
		return
	}
	if !propia {
		errorAlEliminar(w, r, http.StatusNotFound, "Suscripción no encontrada.")
		return
	}
	// la lógica rechaza las de solo lectura (vencidas o de un mes anterior)
	if err := h.suscripciones.Eliminar(r.Context(), id); err != nil {
		estado, mensaje := describirError(err)
		errorAlEliminar(w, r, estado, mensaje)
		return
	}

	todas, err := h.suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		errorInterno(w, err)
		return
	}
	d := datosDelMes(todas, op)
	aviso := views.Aviso{Texto: avisosOK["eliminada"]}
	if len(d.Suscripciones) == 0 {
		w.Header().Set("HX-Retarget", "#lista-suscripciones")
		w.Header().Set("HX-Reswap", "outerHTML")
		render(w, r, http.StatusOK, views.RespuestaListaVacia(d, aviso))
		return
	}
	render(w, r, http.StatusOK, views.RespuestaSuscripcionEliminada(d, aviso))
}

// errorAlEliminar: la fila no se toca y el error se muestra en el aviso
func errorAlEliminar(w http.ResponseWriter, r *http.Request, estado int, mensaje string) {
	estado = cambiarDestinoHTMX(w, r, estado, "#avisos", "outerHTML")
	render(w, r, estado, views.Avisos(avisoError(mensaje), false))
}
