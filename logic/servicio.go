package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// Categorías del catálogo, en el orden en que se cargan
var Categorias = []string{"Streaming", "Música", "Software", "Hogar", "Seguros", "Deportes", "Salud", "Financiero", "Turismo", "Otros"}

// CatalogoInicial son los servicios que se precargan al iniciar la aplicación.
// Los usuarios eligen de este catálogo al crear una suscripción; no se pueden
// agregar ni borrar servicios desde la API.
var CatalogoInicial = map[string][]string{
	"Streaming":  {"Netflix", "HBO Max", "Disney+", "Prime Video", "Paramount+", "Star+", "YouTube Premium"},
	"Música":     {"Spotify", "iTunes", "YouTube Music", "Deezer", "Tidal"},
	"Software":   {"Microsoft 365", "Google One", "iCloud+", "Adobe Creative Cloud", "ChatGPT Plus", "Canva Pro", "Dropbox"},
	"Hogar":      {"Internet / Wifi", "Luz", "Gas", "Agua", "Cable / TV", "Telefonía fija", "Expensas"},
	"Seguros":    {"Seguro de auto", "Seguro de moto", "Seguro de hogar", "Seguro de vida", "Seguro de celular", "Seguro de viaje"},
	"Deportes":   {"Gimnasio", "Pádel", "Fútbol", "Tenis", "Natación", "Yoga", "Crossfit", "Club deportivo"},
	"Salud":      {"OSDE", "Swiss Medical", "Galeno", "Medifé", "Sancor Salud", "Omint"},
	"Financiero": {"Mantenimiento de cuenta bancaria", "Cuota de préstamo", "Tarjeta de crédito"},
	"Turismo":    {"Aerolíneas Plus", "Despegar", "Booking.com", "Airbnb"},
	"Otros":      {"Telefonía celular", "Mercado Libre Meli+", "Rappi Pro", "PedidosYa Plus"},
}

type ServicioServicio struct {
	consultas *sqlc.Queries
}

func NuevoServicioServicio(q *sqlc.Queries) *ServicioServicio {
	return &ServicioServicio{consultas: q}
}

// CargarCatalogoInicial inserta los servicios precargados que todavía no
// existan. Se puede llamar en cada arranque sin duplicar nada.
func (s *ServicioServicio) CargarCatalogoInicial(ctx context.Context) error {
	for _, categoria := range Categorias {
		for _, nombre := range CatalogoInicial[categoria] {
			err := s.consultas.CrearServicioSiNoExiste(ctx, sqlc.CrearServicioSiNoExisteParams{
				Nombre:    nombre,
				Categoria: categoria,
			})
			if err != nil {
				return fmt.Errorf("no se pudo cargar el servicio %q: %w", nombre, err)
			}
		}
	}
	return nil
}

// ObtenerPorID busca un servicio por id_servicio
func (s *ServicioServicio) ObtenerPorID(ctx context.Context, id int32) (sqlc.Servicio, error) {
	srv, err := s.consultas.GetServicio(ctx, id)
	if err == sql.ErrNoRows {
		return sqlc.Servicio{}, fmt.Errorf("servicio no encontrado")
	} else if err != nil {
		return sqlc.Servicio{}, err
	}
	return srv, nil
}

// Listar obtiene la lista de todos los servicios
func (s *ServicioServicio) Listar(ctx context.Context) ([]sqlc.Servicio, error) {
	return s.consultas.ListServicios(ctx)
}

type ManejadorServicio struct {
	servicio *ServicioServicio
}

func NuevoManejadorServicio(s *ServicioServicio) *ManejadorServicio {
	return &ManejadorServicio{servicio: s}
}

// ManejarServicios atiende GET en /servicios. El catálogo es fijo: no se
// pueden crear servicios (POST responde 405).
func (h *ManejadorServicio) ManejarServicios(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.obtenerTodos(w, r)
	default:
		http.Error(w, "Método HTTP no permitido (el catálogo de servicios viene precargado)", http.StatusMethodNotAllowed)
	}
}

// ManejarServicioPorID atiende GET en /servicios/{id}. PUT y DELETE responden 405.
func (h *ManejadorServicio) ManejarServicioPorID(w http.ResponseWriter, r *http.Request) {
	id, ok := h.extraerID(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.obtenerPorID(w, r, id)
	default:
		http.Error(w, "Método HTTP no permitido (el catálogo de servicios viene precargado)", http.StatusMethodNotAllowed)
	}
}

func (h *ManejadorServicio) obtenerTodos(w http.ResponseWriter, r *http.Request) {
	servicios, err := h.servicio.Listar(r.Context())
	if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusOK, servicios)
}

func (h *ManejadorServicio) obtenerPorID(w http.ResponseWriter, r *http.Request, id int32) {
	srv, err := h.servicio.ObtenerPorID(r.Context(), id)
	if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusOK, srv)
}

func (h *ManejadorServicio) extraerID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	partes := strings.Split(r.URL.Path, "/")
	if len(partes) < 3 || partes[len(partes)-1] == "" {
		http.Error(w, "ID de servicio no especificado", http.StatusBadRequest)
		return 0, false
	}

	idNum, err := strconv.Atoi(partes[len(partes)-1])
	if err != nil {
		http.Error(w, "El ID debe ser un valor numérico", http.StatusBadRequest)
		return 0, false
	}

	return int32(idNum), true
}

func (h *ManejadorServicio) responderError(w http.ResponseWriter, err error) {
	if strings.Contains(err.Error(), "no encontrado") {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
}

func (h *ManejadorServicio) responderJSON(w http.ResponseWriter, estado int, datos any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(datos)
}
