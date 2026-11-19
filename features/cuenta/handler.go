// Package cuenta es el slice de "Mi cuenta": el saludo, lo que vence en el
// mes actual y todas las suscripciones activas.
package cuenta

import (
	"net/http"

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
	mux.HandleFunc("GET /cuenta", app.ConSesion(h.cuenta))
}

// cuenta: "Hola, {nombre}" y lo que vence en el mes actual
func (h *handler) cuenta(w http.ResponseWriter, r *http.Request, usr sqlc.Usuario) {
	todas, err := h.app.Suscripciones.ListarPorUsuario(r.Context(), usr.IDUsuario)
	if err != nil {
		plataforma.ErrorInterno(w, err)
		return
	}
	mes := plataforma.MesActual()
	delMes := plataforma.FiltrarPorMes(todas, mes)
	cantidad := 0
	for _, s := range delMes {
		if vista.CuentaParaElGasto(s.Estado) {
			cantidad++
		}
	}

	p := h.app.Pagina(r, "Mi cuenta", "cuenta", &usr)
	p.Aviso = plataforma.AvisoDeLaURL(r)
	plataforma.Render(w, r, http.StatusOK, CuentaPage(p, ResumenMes{
		NombreMes: vista.NombreMes(mes),
		Cantidad:  cantidad,
		Total:     vista.TotalActivo(delMes),
	}, vista.ActivasDe(todas)))
}
