package logic

import (
	"context"
	"database/sql"
	"fmt"

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
