package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// Funciones auxiliares para leer los formularios y los parámetros de la URL,
// y para filtrar las suscripciones por mes y por búsqueda.

func parsearFormulario(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "No se pudieron leer los datos del formulario", http.StatusBadRequest)
		return false
	}
	return true
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
