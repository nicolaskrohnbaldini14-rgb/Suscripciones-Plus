package views

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// Fila es una suscripción del usuario con el nombre y la categoría del servicio
type Fila = sqlc.ListSubscripcionesUsuarioRow

// FormModificar son los valores del formulario para modificar una suscripción
type FormModificar struct {
	Monto         string
	UsuarioCuenta string
	Estado        string
}

// Detalle es la suscripción que se muestra en el recuadro del detalle
type Detalle struct {
	Sub              Fila
	Editable         bool
	Editando         bool
	Eliminando       bool // se pide confirmación antes de eliminar
	Form             FormModificar
	ErrorEditar      string
	PasswordRevelada bool
	Password         *string
	ErrorPassword    string
}

// DiaCalendario es una celda del calendario del mes (Dia = 0: celda vacía)
type DiaCalendario struct {
	Dia     int
	Columna int // 0 = lunes ... 6 = domingo
	Hoy     bool
	Vencen  []Fila
}

// DatosSuscripciones es todo lo que muestra la página "Mis suscripciones"
type DatosSuscripciones struct {
	Mes             string // AAAA-MM
	NombreMes       string
	EsMesActual     bool
	SoloLectura     bool // mes anterior al actual
	MesAnterior     string
	NombreAnterior  string
	MesSiguiente    string
	NombreSiguiente string
	Busqueda        string
	Suscripciones   []Fila // las del mes que coinciden con la búsqueda
	HayEnElMes      bool
	HayAlguna       bool
	Total           string
	Calendario      []DiaCalendario
	Detalle         *Detalle
	Editables       map[int32]bool
}

var nombresMeses = []string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio",
	"agosto", "septiembre", "octubre", "noviembre", "diciembre"}

// NombreMes convierte "2026-10" en "octubre de 2026"
func NombreMes(clave string) string {
	anio, mes, ok := partesMes(clave)
	if !ok {
		return clave
	}
	return nombresMeses[mes-1] + " de " + strconv.Itoa(anio)
}

// SumarMeses("2026-12", 1) devuelve "2027-01"
func SumarMeses(clave string, cantidad int) string {
	anio, mes, ok := partesMes(clave)
	if !ok {
		return clave
	}
	return time.Date(anio, time.Month(mes)+time.Month(cantidad), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
}

func partesMes(clave string) (int, int, bool) {
	t, err := time.Parse("2006-01", clave)
	if err != nil {
		return 0, 0, false
	}
	return t.Year(), int(t.Month()), true
}

// ArmarCalendario arma las celdas del mes (empezando el lunes) y ubica en
// cada día las suscripciones que vencen
func ArmarCalendario(clave string, suscripciones []Fila, hoy time.Time) []DiaCalendario {
	anio, mes, ok := partesMes(clave)
	if !ok {
		return nil
	}
	primero := time.Date(anio, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)
	diasDelMes := primero.AddDate(0, 1, -1).Day()
	columnaDelDia1 := (int(primero.Weekday()) + 6) % 7 // lunes = 0

	porDia := map[int][]Fila{}
	for _, s := range suscripciones {
		porDia[s.FechaVencimiento.Day()] = append(porDia[s.FechaVencimiento.Day()], s)
	}

	esMesActual := hoy.Year() == anio && int(hoy.Month()) == mes
	var celdas []DiaCalendario
	for i := 0; i < columnaDelDia1; i++ {
		celdas = append(celdas, DiaCalendario{Columna: i})
	}
	for dia := 1; dia <= diasDelMes; dia++ {
		celdas = append(celdas, DiaCalendario{
			Dia:     dia,
			Columna: (columnaDelDia1 + dia - 1) % 7,
			Hoy:     esMesActual && hoy.Day() == dia,
			Vencen:  porDia[dia],
		})
	}
	return celdas
}

// CuentaParaElGasto: las activas y los períodos que ya vencieron y se
// renovaron (se pagaron); no las pausadas
func CuentaParaElGasto(estado string) bool {
	return estado == "Activa" || estado == "Vencida"
}

// TotalActivo suma el monto de las suscripciones que cuentan para el gasto
// (es el gasto de un mes anterior: todo lo que venció ese mes)
func TotalActivo(suscripciones []Fila) string {
	return sumarMontos(suscripciones, func(s Fila) bool { return CuentaParaElGasto(s.Estado) })
}

// GastoPendiente suma lo que falta pagar desde hoy: las suscripciones activas
// que todavía no vencieron (lo que ya venció en el mes no se cuenta). Es el
// gasto que se muestra en el mes actual y en los siguientes.
func GastoPendiente(suscripciones []Fila, hoy time.Time) string {
	dia := time.Date(hoy.Year(), hoy.Month(), hoy.Day(), 0, 0, 0, 0, time.UTC)
	return sumarMontos(suscripciones, func(s Fila) bool {
		return s.Estado == "Activa" && !s.FechaVencimiento.Before(dia)
	})
}

func sumarMontos(suscripciones []Fila, cuenta func(Fila) bool) string {
	total := 0.0
	for _, s := range suscripciones {
		if !cuenta(s) {
			continue
		}
		if monto, err := strconv.ParseFloat(s.Monto, 64); err == nil {
			total += monto
		}
	}
	return "$ " + formatearNumero(total)
}

// textoEstado: un período "Vencida" ya se renovó
func textoEstado(estado string) string {
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

// UrlEliminar lleva al recuadro que pide confirmación antes de eliminar una
// suscripción (?ver=id&eliminar=1)
func UrlEliminar(mes, busqueda string, id int32) string {
	return UrlSuscripciones(mes, busqueda, id, false) + "&eliminar=1"
}

func valorCuenta(cuenta *string) string {
	if cuenta == nil {
		return ""
	}
	return *cuenta
}

// mayusculaInicial: "octubre de 2026" -> "Octubre de 2026"
func mayusculaInicial(texto string) string {
	if texto == "" {
		return texto
	}
	r := []rune(texto)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}
