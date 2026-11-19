package vista

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// Datos y funciones de presentación de las suscripciones que usan varios
// slices (Mi cuenta, Nueva suscripción y Mis suscripciones).

// Fila es una suscripción del usuario con el nombre y la categoría del servicio
type Fila = sqlc.ListSubscripcionesUsuarioRow

// Activas son todas las suscripciones activas del usuario (de cualquier mes),
// ordenadas por vencimiento
type Activas struct {
	Lista []Fila
}

// ActivasDe devuelve todas las suscripciones activas (de cualquier mes),
// ordenadas por vencimiento: el panel de "Mi cuenta" y de "Nueva suscripción"
func ActivasDe(todas []Fila) Activas {
	var activas []Fila
	for _, s := range todas {
		if s.Estado == "Activa" {
			activas = append(activas, s)
		}
	}
	sort.SliceStable(activas, func(i, j int) bool {
		return activas[i].FechaVencimiento.Before(activas[j].FechaVencimiento)
	})
	return Activas{Lista: activas}
}

var nombresMeses = []string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio",
	"agosto", "septiembre", "octubre", "noviembre", "diciembre"}

// NombreMes convierte "2026-10" en "octubre de 2026"
func NombreMes(clave string) string {
	anio, mes, ok := PartesMes(clave)
	if !ok {
		return clave
	}
	return nombresMeses[mes-1] + " de " + strconv.Itoa(anio)
}

// SumarMeses("2026-12", 1) devuelve "2027-01"
func SumarMeses(clave string, cantidad int) string {
	anio, mes, ok := PartesMes(clave)
	if !ok {
		return clave
	}
	return time.Date(anio, time.Month(mes)+time.Month(cantidad), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
}

// PartesMes separa "2026-10" en año y mes
func PartesMes(clave string) (int, int, bool) {
	t, err := time.Parse("2006-01", clave)
	if err != nil {
		return 0, 0, false
	}
	return t.Year(), int(t.Month()), true
}

// CuentaParaElGasto: las activas y los períodos que ya vencieron y se
// renovaron (se pagaron); no las pausadas
func CuentaParaElGasto(estado string) bool {
	return estado == "Activa" || estado == "Vencida"
}

// TotalActivo suma el monto de las suscripciones que cuentan para el gasto
// (es el gasto de un mes anterior: todo lo que venció ese mes)
func TotalActivo(suscripciones []Fila) string {
	return SumarMontos(suscripciones, func(s Fila) bool { return CuentaParaElGasto(s.Estado) })
}

// SumarMontos suma (y formatea) el monto de las suscripciones que cumplen
// la condición
func SumarMontos(suscripciones []Fila, cuenta func(Fila) bool) string {
	total := 0.0
	for _, s := range suscripciones {
		if !cuenta(s) {
			continue
		}
		if monto, err := strconv.ParseFloat(s.Monto, 64); err == nil {
			total += monto
		}
	}
	return "$ " + FormatearNumero(total)
}

// TextoEstado: un período "Vencida" ya se renovó
func TextoEstado(estado string) string {
	if estado == "Vencida" {
		return "Vencida (renovada)"
	}
	return estado
}

// UrlSuscripciones arma la dirección de la lista conservando el mes y la
// búsqueda; ver > 0 abre el detalle de esa suscripción
func UrlSuscripciones(mes, busqueda string, ver int32, editar bool) string {
	v := url.Values{}
	if mes != "" {
		v.Set("mes", mes)
	}
	if busqueda != "" {
		v.Set("q", busqueda)
	}
	if ver > 0 {
		v.Set("ver", strconv.Itoa(int(ver)))
	}
	if editar {
		v.Set("editar", "1")
	}
	if len(v) == 0 {
		return "/cuenta/suscripciones"
	}
	return "/cuenta/suscripciones?" + v.Encode()
}

// MayusculaInicial: "octubre de 2026" -> "Octubre de 2026"
func MayusculaInicial(texto string) string {
	if texto == "" {
		return texto
	}
	r := []rune(texto)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}
