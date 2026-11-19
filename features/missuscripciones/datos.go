package missuscripciones

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"tp2-krohn-trapani-cicopiedi/logic"
	"tp2-krohn-trapani-cicopiedi/plataforma"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

// Datos (DTOs) que muestra la página "Mis suscripciones" y las funciones que
// los arman. Son solo de este slice.

// FormModificar son los valores del formulario para modificar una suscripción
type FormModificar struct {
	Monto         string
	UsuarioCuenta string
	Estado        string
}

// Detalle es la suscripción que se muestra en el recuadro del detalle
type Detalle struct {
	Sub              vista.Fila
	Editable         bool
	Editando         bool
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
	Vencen  []vista.Fila
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
	Suscripciones   []vista.Fila // las del mes que coinciden con la búsqueda
	HayEnElMes      bool
	HayAlguna       bool
	Total           string
	Calendario      []DiaCalendario
	Detalle         *Detalle
	Editables       map[int32]bool
}

// ArmarCalendario arma las celdas del mes (empezando el lunes) y ubica en
// cada día las suscripciones que vencen
func ArmarCalendario(clave string, suscripciones []vista.Fila, hoy time.Time) []DiaCalendario {
	anio, mes, ok := vista.PartesMes(clave)
	if !ok {
		return nil
	}
	primero := time.Date(anio, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)
	diasDelMes := primero.AddDate(0, 1, -1).Day()
	columnaDelDia1 := (int(primero.Weekday()) + 6) % 7 // lunes = 0

	porDia := map[int][]vista.Fila{}
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

// GastoPendiente suma lo que falta pagar desde hoy: las suscripciones activas
// que todavía no vencieron (lo que ya venció en el mes no se cuenta). Es el
// gasto que se muestra en el mes actual y en los siguientes.
func GastoPendiente(suscripciones []vista.Fila, hoy time.Time) string {
	dia := time.Date(hoy.Year(), hoy.Month(), hoy.Day(), 0, 0, 0, 0, time.UTC)
	return vista.SumarMontos(suscripciones, func(s vista.Fila) bool {
		return s.Estado == "Activa" && !s.FechaVencimiento.Before(dia)
	})
}

// EsEditable: ni de un mes anterior ni un período ya renovado
func EsEditable(s vista.Fila) bool {
	return !logic.EsSoloLectura(s.Estado, s.FechaVencimiento)
}

// buscar filtra por servicio, categoría o usuario de la cuenta (sin
// distinguir mayúsculas)
func buscar(suscripciones []vista.Fila, texto string) []vista.Fila {
	texto = strings.ToLower(strings.TrimSpace(texto))
	if texto == "" {
		return suscripciones
	}
	var coinciden []vista.Fila
	for _, s := range suscripciones {
		for _, campo := range []string{s.ServicioNombre, s.ServicioCategoria, plataforma.Valor(s.UsuarioCuenta)} {
			if strings.Contains(strings.ToLower(campo), texto) {
				coinciden = append(coinciden, s)
				break
			}
		}
	}
	return coinciden
}

// UrlConfirmarEliminar es la ruta que devuelve el aviso para confirmar el
// borrado (hx-get del botón "Eliminar"); con cancelar=true devuelve el aviso
// vacío (botón "Cancelar"). Lleva el mes y la búsqueda para el DELETE.
func UrlConfirmarEliminar(mes, busqueda string, id int32, cancelar bool) string {
	v := url.Values{}
	if mes != "" {
		v.Set("mes", mes)
	}
	if busqueda != "" {
		v.Set("q", busqueda)
	}
	if cancelar {
		v.Set("cancelar", "1")
	}
	ruta := "/cuenta/suscripciones/" + strconv.Itoa(int(id)) + "/confirmar-eliminar"
	if len(v) == 0 {
		return ruta
	}
	return ruta + "?" + v.Encode()
}

// UrlBorrar es la ruta del DELETE de una suscripción (hx-delete). Lleva el
// mes y la búsqueda de la lista para recalcular el gasto y el calendario.
func UrlBorrar(mes, busqueda string, id int32) string {
	v := url.Values{}
	if mes != "" {
		v.Set("mes", mes)
	}
	if busqueda != "" {
		v.Set("q", busqueda)
	}
	ruta := "/cuenta/suscripciones/" + strconv.Itoa(int(id))
	if len(v) == 0 {
		return ruta
	}
	return ruta + "?" + v.Encode()
}
