package views

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

func renderizar(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("Error al renderizar el componente: %v", err)
	}
	return buf.String()
}

func verificarContiene(t *testing.T, html string, esperados ...string) {
	t.Helper()
	for _, e := range esperados {
		if !strings.Contains(html, e) {
			t.Errorf("La página debería contener %q", e)
		}
	}
}

func verificarNoContiene(t *testing.T, html string, prohibidos ...string) {
	t.Helper()
	for _, p := range prohibidos {
		if strings.Contains(html, p) {
			t.Errorf("La página no debería contener %q", p)
		}
	}
}

func fecha(anio int, mes time.Month, dia int) time.Time {
	return time.Date(anio, mes, dia, 0, 0, 0, 0, time.UTC)
}

var catalogo = []sqlc.Servicio{
	{IDServicio: 1, Nombre: "Netflix", Categoria: "Streaming"},
	{IDServicio: 2, Nombre: "Spotify", Categoria: "Música"},
	{IDServicio: 3, Nombre: "Pádel", Categoria: "Deportes"},
}

func TestHomePage_SinSesion(t *testing.T) {
	html := renderizar(t, HomePage(Pagina{Titulo: "Inicio", Clave: "inicio", Servicios: catalogo}))

	verificarContiene(t, html,
		"<!doctype html>",
		`href="/static/styles.css"`,
		`href="/login"`,
		`href="/registro"`,
		// panel de servicios agrupado por categoría, con su logo
		"panel-servicios",
		"Streaming", "Música", "Deportes",
		`class="icono-categoria"`,
		"Netflix",
	)
	// sin sesión los servicios no son links para suscribirse
	verificarNoContiene(t, html, "/cuenta/suscripciones/nueva?servicio=", "Cerrar sesión", "<script")
}

func TestHomePage_ConSesion(t *testing.T) {
	html := renderizar(t, HomePage(Pagina{Clave: "inicio", Sesion: &Sesion{Nombre: "Nico"}, Servicios: catalogo}))

	verificarContiene(t, html,
		"Hola, Nico",
		"Cerrar sesión",
		`href="/cuenta"`,
		`href="/cuenta/suscripciones/nueva?servicio=1"`,
	)
}

func TestCuentaPage(t *testing.T) {
	html := renderizar(t, CuentaPage(Pagina{Clave: "cuenta", Sesion: &Sesion{Nombre: "Nico"}},
		ResumenMes{NombreMes: "octubre de 2026", Cantidad: 1, Total: "$ 2.500,00"}))

	verificarContiene(t, html,
		"Hola, Nico",
		"En octubre de 2026 vence 1 suscripción · $ 2.500,00",
		`href="/cuenta/suscripciones"`,
		`href="/cuenta/suscripciones/nueva"`,
	)
}

func filaDePrueba() Fila {
	cuenta := "martinagym"
	return Fila{
		IDSuscripcion:     5,
		Monto:             "32000.00",
		FechaInicio:       fecha(2026, 10, 1),
		FechaVencimiento:  fecha(2026, 10, 28),
		UsuarioCuenta:     &cuenta,
		Estado:            "Activa",
		ServicioNombre:    "Gimnasio",
		ServicioCategoria: "Deportes",
	}
}

func TestMisSuscripcionesPage_Editable(t *testing.T) {
	s := filaDePrueba()
	d := DatosSuscripciones{
		Mes: "2026-10", NombreMes: "octubre de 2026", EsMesActual: true,
		Suscripciones: []Fila{s}, HayEnElMes: true, HayAlguna: true,
		Total: "$ 32.000,00", Editables: map[int32]bool{5: true},
		Calendario: ArmarCalendario("2026-10", []Fila{s}, fecha(2026, 10, 29)),
	}
	html := renderizar(t, MisSuscripcionesPage(Pagina{Clave: "suscripciones", Sesion: &Sesion{Nombre: "Nico"}}, d))

	verificarContiene(t, html,
		"Gimnasio — $ 32.000,00",
		`<span class="estado-fijo">Activa</span>`,
		`action="/cuenta/suscripciones/5/eliminar"`,
		"Modificar",
		`name="q"`,
		"Gasto mensual (activo): $ 32.000,00",
		"Vence el 28:",
	)
	// la lista no muestra la cuenta ni las fechas (están en el detalle) y el
	// estado se cambia desde "Modificar", no desde la lista
	verificarNoContiene(t, html, "martinagym", "<script", `name="estado"`, "Cancelada")
}

func TestDetalle_ModificarConEstado(t *testing.T) {
	s := filaDePrueba()
	d := DatosSuscripciones{
		Mes: "2026-10", NombreMes: "octubre de 2026", EsMesActual: true,
		Suscripciones: []Fila{s}, HayEnElMes: true, HayAlguna: true,
		Editables: map[int32]bool{5: true},
		Detalle: &Detalle{Sub: s, Editable: true, Editando: true,
			Form: FormModificar{Monto: "32000", UsuarioCuenta: "martinagym", Estado: "Pausada"}},
	}
	html := renderizar(t, MisSuscripcionesPage(Pagina{Clave: "suscripciones", Sesion: &Sesion{Nombre: "Nico"}}, d))

	verificarContiene(t, html,
		`action="/cuenta/suscripciones/5/modificar"`,
		`<select name="estado">`,
		`<option value="Activa">Activa</option>`,
		`<option value="Pausada" selected>Pausada</option>`,
	)
	// no existe el estado Cancelada: para dejar una suscripción se la elimina
	verificarNoContiene(t, html, "Cancelada")
}

func TestMisSuscripcionesPage_SoloLecturaYDetalle(t *testing.T) {
	s := filaDePrueba()
	s.Estado = "Vencida"
	password := "gym2026!"
	d := DatosSuscripciones{
		Mes: "2026-09", NombreMes: "septiembre de 2026", SoloLectura: true,
		Suscripciones: []Fila{s}, HayEnElMes: true, HayAlguna: true,
		Editables: map[int32]bool{5: false},
		Detalle:   &Detalle{Sub: s, PasswordRevelada: true, Password: &password},
	}
	html := renderizar(t, MisSuscripcionesPage(Pagina{Clave: "suscripciones", Sesion: &Sesion{Nombre: "Nico"}}, d))

	verificarContiene(t, html,
		"solo de lectura",
		"Vencida (renovada)",
		"martinagym",
		"gym2026!",
		`class="detalle-fondo"`,
	)
	verificarNoContiene(t, html, `/estado"`, `/eliminar"`, `/modificar"`)
}

func TestNuevaSuscripcionPage(t *testing.T) {
	p := Pagina{Clave: "nueva-suscripcion", Sesion: &Sesion{Nombre: "Nico"}, Servicios: catalogo,
		Catalogo: Catalogo{Abierto: true, Elegir: true, IDActual: 3, CerrarURL: "/cuenta/suscripciones/nueva?servicio=3"}}
	html := renderizar(t, NuevaSuscripcionPage(p, FormSuscripcion{IDServicio: 3, Monto: "15000"}))

	verificarContiene(t, html,
		`<optgroup label="Deportes">`,
		`<option value="3" selected>Pádel</option>`,
		`value="15000"`,
		// el catálogo abierto en modo "elegir" con el servicio actual resaltado
		"panel-servicios abierto modo-elegir",
		"panel-servicio elegido",
	)
	// se crea siempre activa: no se elige el estado ni el usuario
	verificarNoContiene(t, html, `name="estado"`, `name="id_usuario"`)
}

func TestArmarCalendario(t *testing.T) {
	celdas := ArmarCalendario("2026-10", []Fila{filaDePrueba()}, fecha(2026, 10, 29))

	// octubre de 2026 empieza un jueves: 3 celdas vacías antes del día 1
	if celdas[3].Dia != 1 || celdas[2].Dia != 0 {
		t.Fatalf("el 1 de octubre de 2026 debería caer jueves, celdas: %+v", celdas[:4])
	}
	dia28 := celdas[3+27]
	if dia28.Dia != 28 || len(dia28.Vencen) != 1 {
		t.Errorf("el día 28 debería tener 1 vencimiento: %+v", dia28)
	}
	if !celdas[3+28].Hoy {
		t.Error("el 29 debería estar marcado como hoy")
	}
}

func TestMeses(t *testing.T) {
	if NombreMes("2026-10") != "octubre de 2026" {
		t.Errorf("NombreMes = %q", NombreMes("2026-10"))
	}
	if SumarMeses("2026-12", 1) != "2027-01" || SumarMeses("2026-01", -1) != "2025-12" {
		t.Error("SumarMeses no cruza bien los años")
	}
}

func TestItemSuscripcion_EscapaHTML(t *testing.T) {
	s := filaDePrueba()
	s.ServicioNombre = "<script>alert(1)</script>"
	html := renderizar(t, ItemSuscripcion(DatosSuscripciones{Editables: map[int32]bool{}}, s))
	verificarNoContiene(t, html, "<script>")
}

func TestFormatearNumero(t *testing.T) {
	casos := map[float64]string{
		0:          "0,00",
		999.5:      "999,50",
		1500:       "1.500,00",
		1234567.89: "1.234.567,89",
		-2500:      "-2.500,00",
	}
	for valor, esperado := range casos {
		if obtenido := formatearNumero(valor); obtenido != esperado {
			t.Errorf("formatearNumero(%v) = %q, se esperaba %q", valor, obtenido, esperado)
		}
	}
}

func TestGastoPendiente(t *testing.T) {
	fila := func(estado, monto string, vence time.Time) Fila {
		return Fila{Estado: estado, Monto: monto, FechaVencimiento: vence}
	}
	subs := []Fila{
		fila("Activa", "1000", fecha(2026, 11, 2)),   // ya venció este mes: no se cuenta
		fila("Activa", "2000", fecha(2026, 11, 4)),   // vence hoy: falta pagarla
		fila("Activa", "3000", fecha(2026, 11, 20)),  // falta pagarla
		fila("Pausada", "4000", fecha(2026, 11, 25)), // pausada: no suma
		fila("Vencida", "5000", fecha(2026, 11, 1)),  // período ya renovado (pagado)
	}
	hoy := time.Date(2026, 11, 4, 15, 30, 0, 0, time.Local)

	if obtenido := GastoPendiente(subs, hoy); obtenido != "$ 5.000,00" {
		t.Errorf("GastoPendiente = %s, se esperaba $ 5.000,00 (solo lo que falta pagar)", obtenido)
	}
	// en un mes anterior se cuenta todo lo que venció (activas y renovadas)
	if obtenido := TotalActivo(subs); obtenido != "$ 11.000,00" {
		t.Errorf("TotalActivo = %s, se esperaba $ 11.000,00", obtenido)
	}
}
