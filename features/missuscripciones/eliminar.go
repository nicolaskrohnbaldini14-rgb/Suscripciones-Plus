package missuscripciones

import (
	"net/http"
	"strings"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/logic"
	"tp2-krohn-trapani-cicopiedi/plataforma"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

// confirmarEliminar maneja GET /cuenta/suscripciones/{id}/confirmar-eliminar
// (TP6): devuelve el aviso que pregunta si se quiere eliminar la suscripción
// (en lugar de la ventana del navegador), o el aviso vacío si se canceló.
func (h *handler) confirmarEliminar(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	q := r.URL.Query()
	if q.Get("cancelar") == "1" {
		plataforma.Render(w, r, http.StatusOK, vista.Avisos(vista.Aviso{}, false))
		return
	}

	id := plataforma.LeerEntero(r.PathValue("id"))
	todas, err := h.app.Suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return
	}
	for _, s := range todas {
		if s.IDSuscripcion != id {
			continue
		}
		// las de solo lectura (vencidas o de un mes anterior) no se borran
		if logic.EsSoloLectura(s.Estado, s.FechaVencimiento) {
			estado, mensaje := plataforma.DescribirError(logic.ErrSoloLectura)
			errorAlEliminar(w, r, estado, mensaje)
			return
		}
		plataforma.Render(w, r, http.StatusOK, ConfirmarEliminar(plataforma.MesValido(q.Get("mes")), strings.TrimSpace(q.Get("q")), s))
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
func (h *handler) eliminar(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	q := r.URL.Query()
	op := opcionesLista{Mes: plataforma.MesValido(q.Get("mes")), Busqueda: strings.TrimSpace(q.Get("q"))}

	id := plataforma.LeerEntero(r.PathValue("id"))
	propia, err := h.app.Suscripciones.PerteneceA(r.Context(), id, usr.IDUsuario)
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return
	}
	if !propia {
		errorAlEliminar(w, r, http.StatusNotFound, "Suscripción no encontrada.")
		return
	}
	// la lógica rechaza las de solo lectura (vencidas o de un mes anterior)
	if err := h.app.Suscripciones.Eliminar(r.Context(), id); err != nil {
		estado, mensaje := plataforma.DescribirError(err)
		errorAlEliminar(w, r, estado, mensaje)
		return
	}
	plataforma.SuscripcionesEliminadas.Inc()

	todas, err := h.app.Suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return
	}
	d := datosDelMes(todas, op)
	aviso := vista.Aviso{Texto: plataforma.AvisosOK["eliminada"]}
	if len(d.Suscripciones) == 0 {
		w.Header().Set("HX-Retarget", "#lista-suscripciones")
		w.Header().Set("HX-Reswap", "outerHTML")
		plataforma.Render(w, r, http.StatusOK, RespuestaListaVacia(d, aviso))
		return
	}
	plataforma.Render(w, r, http.StatusOK, RespuestaSuscripcionEliminada(d, aviso))
}

// errorAlEliminar: la fila no se toca y el error se muestra en el aviso
func errorAlEliminar(w http.ResponseWriter, r *http.Request, estado int, mensaje string) {
	estado = plataforma.CambiarDestinoHTMX(w, r, estado, "#avisos", "outerHTML")
	plataforma.Render(w, r, estado, vista.Avisos(plataforma.AvisoError(mensaje), false))
}
