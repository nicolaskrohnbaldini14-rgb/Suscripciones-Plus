package handlers

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/logic"
	"tp2-krohn-trapani-cicopiedi/views"
)

// Web reúne los handlers que renderizan HTML con templ. Reemplazan a los
// handlers JSON del TP4: la vista se genera en el servidor y no hay JavaScript.
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

// ---------------------------------------------------------------------------
// Sesión y datos comunes de las páginas
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Páginas públicas
// ---------------------------------------------------------------------------

func (h *Web) home(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, views.HomePage(h.pagina(r, "Inicio", "inicio", h.usuarioActual(r))))
}

func (h *Web) formLogin(w http.ResponseWriter, r *http.Request) {
	if h.usuarioActual(r) != nil {
		http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
		return
	}
	render(w, r, http.StatusOK, views.LoginPage(h.pagina(r, "Iniciar sesión", "login", nil), ""))
}

func (h *Web) login(w http.ResponseWriter, r *http.Request) {
	if !parsearFormulario(w, r) {
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))

	usr, err := h.usuarios.Autenticar(r.Context(), email, r.FormValue("password"))
	if err != nil {
		estado, mensaje := describirError(err)
		p := h.pagina(r, "Iniciar sesión", "login", nil)
		p.Aviso = avisoError(mensaje)
		render(w, r, estado, views.LoginPage(p, email))
		return
	}

	h.iniciarSesionYRedirigir(w, r, usr, "/cuenta")
}

func (h *Web) formRegistro(w http.ResponseWriter, r *http.Request) {
	if h.usuarioActual(r) != nil {
		http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
		return
	}
	render(w, r, http.StatusOK, views.RegistroPage(h.pagina(r, "Registrarse", "registro", nil), views.DatosRegistro{}))
}

// registrar crea la cuenta y deja la sesión iniciada
func (h *Web) registrar(w http.ResponseWriter, r *http.Request) {
	if !parsearFormulario(w, r) {
		return
	}
	datos := views.DatosRegistro{
		Nombre: strings.TrimSpace(r.FormValue("nombre")),
		Email:  strings.TrimSpace(r.FormValue("email")),
	}

	usr, err := h.usuarios.Crear(r.Context(), sqlc.CreateUsuarioParams{
		Nombre:   datos.Nombre,
		Email:    datos.Email,
		Password: r.FormValue("password"),
	})
	if err != nil {
		estado, mensaje := describirError(err)
		p := h.pagina(r, "Registrarse", "registro", nil)
		p.Aviso = avisoError(mensaje)
		render(w, r, estado, views.RegistroPage(p, datos))
		return
	}

	h.iniciarSesionYRedirigir(w, r, usr, "/cuenta?ok=bienvenida")
}

func (h *Web) iniciarSesionYRedirigir(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario, destino string) {
	if err := h.sesiones.Iniciar(w, usr.IDUsuario); err != nil {
		errorInterno(w, err)
		return
	}
	http.Redirect(w, r, destino, http.StatusSeeOther)
}

func (h *Web) logout(w http.ResponseWriter, r *http.Request) {
	h.sesiones.Cerrar(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ---------------------------------------------------------------------------
// Cuenta
// ---------------------------------------------------------------------------

// cuenta: "Hola, {nombre}" y lo que vence en el mes actual
func (h *Web) cuenta(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	todas, err := h.suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		errorInterno(w, err)
		return
	}
	mes := mesActual()
	delMes := filtrarPorMes(todas, mes)
	cantidad := 0
	for _, s := range delMes {
		if views.CuentaParaElGasto(s.Estado) {
			cantidad++
		}
	}

	p := h.pagina(r, "Mi cuenta", "cuenta", &usr)
	p.Aviso = avisoDeLaURL(r)
	render(w, r, http.StatusOK, views.CuentaPage(p, views.ResumenMes{
		NombreMes: views.NombreMes(mes),
		Cantidad:  cantidad,
		Total:     views.TotalActivo(delMes),
	}))
}

// ---------------------------------------------------------------------------
// Mis suscripciones
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Auxiliares
// ---------------------------------------------------------------------------

func render(w http.ResponseWriter, r *http.Request, estado int, componente templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(estado)
	if err := componente.Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar la página: %v", err)
	}
}

func parsearFormulario(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "No se pudieron leer los datos del formulario", http.StatusBadRequest)
		return false
	}
	return true
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

var formatoMes = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func mesActual() string {
	return time.Now().Format("2006-01")
}

// mesValido devuelve el mes pedido (AAAA-MM) o, si no es válido, el actual
func mesValido(mes string) string {
	if formatoMes.MatchString(mes) {
		return mes
	}
	return mesActual()
}

func filtrarPorMes(suscripciones []sqlc.ListSubscripcionesUsuarioRow, mes string) []sqlc.ListSubscripcionesUsuarioRow {
	var delMes []sqlc.ListSubscripcionesUsuarioRow
	for _, s := range suscripciones {
		if s.FechaVencimiento.Format("2006-01") == mes {
			delMes = append(delMes, s)
		}
	}
	return delMes
}

// buscar filtra por servicio, categoría o usuario de la cuenta (sin
// distinguir mayúsculas)
func buscar(suscripciones []sqlc.ListSubscripcionesUsuarioRow, texto string) []sqlc.ListSubscripcionesUsuarioRow {
	texto = strings.ToLower(strings.TrimSpace(texto))
	if texto == "" {
		return suscripciones
	}
	var coinciden []sqlc.ListSubscripcionesUsuarioRow
	for _, s := range suscripciones {
		for _, campo := range []string{s.ServicioNombre, s.ServicioCategoria, valor(s.UsuarioCuenta)} {
			if strings.Contains(strings.ToLower(campo), texto) {
				coinciden = append(coinciden, s)
				break
			}
		}
	}
	return coinciden
}

func leerEntero(valor string) int32 {
	n, err := strconv.Atoi(strings.TrimSpace(valor))
	if err != nil {
		return 0
	}
	return int32(n)
}

// leerFecha interpreta el formato de <input type="date"> (AAAA-MM-DD)
func leerFecha(valor string) time.Time {
	fecha, err := time.Parse("2006-01-02", strings.TrimSpace(valor))
	if err != nil {
		return time.Time{}
	}
	return fecha
}

// texto devuelve un puntero (para los campos opcionales); la lógica guarda
// los vacíos como NULL
func texto(valor string) *string {
	return &valor
}

func valor(puntero *string) string {
	if puntero == nil {
		return ""
	}
	return *puntero
}
