package handlers

import (
	"net/http"
	"sort"
	"strings"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/views"
)

// Handlers de los usuarios: inicio, login, registro, logout y la cuenta.

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

	// todas las activas, de cualquier mes, ordenadas por vencimiento
	var activas []sqlc.ListSubscripcionesUsuarioRow
	for _, s := range todas {
		if s.Estado == "Activa" {
			activas = append(activas, s)
		}
	}
	sort.SliceStable(activas, func(i, j int) bool {
		return activas[i].FechaVencimiento.Before(activas[j].FechaVencimiento)
	})

	p := h.pagina(r, "Mi cuenta", "cuenta", &usr)
	p.Aviso = avisoDeLaURL(r)
	render(w, r, http.StatusOK, views.CuentaPage(p, views.ResumenMes{
		NombreMes: views.NombreMes(mes),
		Cantidad:  cantidad,
		Total:     views.TotalActivo(delMes),
	}, views.Activas{Lista: activas, Total: views.TotalActivo(activas)}))
}
