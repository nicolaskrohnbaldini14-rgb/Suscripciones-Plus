package missuscripciones

import (
	"net/http"
	"strings"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/plataforma"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

// modificar cambia el monto, el usuario, (si se escribe) la contraseña de
// la cuenta y el estado (activa o pausada); si falla, vuelve a mostrar el
// detalle con el error
func (h *handler) modificar(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !plataforma.ParsearFormulario(w, r) {
		return
	}
	op := opcionesDelFormulario(r)
	sub, ok := h.suscripcionDelUsuario(w, r, usr, op)
	if !ok {
		return
	}

	form := FormModificar{
		Monto:         strings.TrimSpace(r.FormValue("monto")),
		UsuarioCuenta: strings.TrimSpace(r.FormValue("usuario_cuenta")),
		Estado:        r.FormValue("estado"),
	}
	// si el formulario no trae el estado, se conserva el que tenía
	if form.Estado == "" {
		form.Estado = sub.Estado
	}
	_, err := h.app.Suscripciones.Actualizar(r.Context(), sub.IDSuscripcion, sqlc.UpdateSubscripcionParams{
		Monto:            form.Monto,
		FechaVencimiento: sub.FechaVencimiento,
		Estado:           form.Estado,
		UsuarioCuenta:    plataforma.Texto(form.UsuarioCuenta),
		PasswordCuenta:   plataforma.Texto(r.FormValue("password_cuenta")),
	})
	if err != nil {
		estado, mensaje := plataforma.DescribirError(err)
		op.Ver = sub.IDSuscripcion
		op.FormEditar = &form
		op.ErrorEditar = mensaje
		h.renderSuscripciones(w, r, usr, estado, op)
		return
	}
	redirigirALista(w, r, op, sub.IDSuscripcion, "modificada")
}

func redirigirALista(w http.ResponseWriter, r *http.Request, op opcionesLista, ver int32, aviso string) {
	destino := vista.UrlSuscripciones(op.Mes, op.Busqueda, ver, false)
	separador := "?"
	if strings.Contains(destino, "?") {
		separador = "&"
	}
	http.Redirect(w, r, destino+separador+"ok="+aviso, http.StatusSeeOther)
}

// revelarPassword muestra la contraseña de la cuenta en el detalle, solo si
// el usuario confirma su propia contraseña (no se redirige: no queda en la URL)
func (h *handler) revelarPassword(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	if !plataforma.ParsearFormulario(w, r) {
		return
	}
	op := opcionesDelFormulario(r)
	sub, ok := h.suscripcionDelUsuario(w, r, usr, op)
	if !ok {
		return
	}
	op.Ver = sub.IDSuscripcion

	password, err := h.app.Suscripciones.RevelarPasswordCuenta(r.Context(), sub.IDSuscripcion, usr.IDUsuario, r.FormValue("password"))
	if err != nil {
		estado, mensaje := plataforma.DescribirError(err)
		op.ErrorPassword = mensaje
		h.renderSuscripciones(w, r, usr, estado, op)
		return
	}
	op.PasswordRevelada = true
	op.Password = password
	h.renderSuscripciones(w, r, usr, http.StatusOK, op)
}
