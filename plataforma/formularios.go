package plataforma

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// Funciones auxiliares para leer los formularios y los parámetros de la URL,
// que usan varios slices.

func ParsearFormulario(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "No se pudieron leer los datos del formulario", http.StatusBadRequest)
		return false
	}
	return true
}

var formatoMes = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func MesActual() string {
	return time.Now().Format("2006-01")
}

// MesValido devuelve el mes pedido (AAAA-MM) o, si no es válido, el actual
func MesValido(mes string) string {
	if formatoMes.MatchString(mes) {
		return mes
	}
	return MesActual()
}

// FiltrarPorMes deja las suscripciones que vencen en el mes (AAAA-MM)
func FiltrarPorMes(suscripciones []sqlc.ListSubscripcionesUsuarioRow, mes string) []sqlc.ListSubscripcionesUsuarioRow {
	var delMes []sqlc.ListSubscripcionesUsuarioRow
	for _, s := range suscripciones {
		if s.FechaVencimiento.Format("2006-01") == mes {
			delMes = append(delMes, s)
		}
	}
	return delMes
}

func LeerEntero(valor string) int32 {
	n, err := strconv.Atoi(strings.TrimSpace(valor))
	if err != nil {
		return 0
	}
	return int32(n)
}

// LeerFecha interpreta el formato de <input type="date"> (AAAA-MM-DD)
func LeerFecha(valor string) time.Time {
	fecha, err := time.Parse("2006-01-02", strings.TrimSpace(valor))
	if err != nil {
		return time.Time{}
	}
	return fecha
}

// Texto devuelve un puntero (para los campos opcionales); la lógica guarda
// los vacíos como NULL
func Texto(valor string) *string {
	return &valor
}

func Valor(puntero *string) string {
	if puntero == nil {
		return ""
	}
	return *puntero
}
