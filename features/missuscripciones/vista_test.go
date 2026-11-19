package missuscripciones

import (
	"strings"
	"testing"
	"time"

	"tp2-krohn-trapani-cicopiedi/plataforma/pruebas"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

func TestMisSuscripcionesPage_Editable(t *testing.T) {
	s := pruebas.Fila()
	d := DatosSuscripciones{
		Mes: "2026-10", NombreMes: "octubre de 2026", EsMesActual: true,
		Suscripciones: []vista.Fila{s}, HayEnElMes: true, HayAlguna: true,
		Total: "$ 32.000,00", Editables: map[int32]bool{5: true},
		Calendario: ArmarCalendario("2026-10", []vista.Fila{s}, pruebas.Fecha(2026, 10, 29)),
	}
	html := pruebas.Renderizar(t, MisSuscripcionesPage(vista.Pagina{Clave: "suscripciones", Sesion: &vista.Sesion{Nombre: "Nico"}}, d))

	pruebas.VerificarContiene(t, html,
		"Gimnasio — $ 32.000,00",
		`<span class="estado-fijo">Activa</span>`,
		// TP6: "Eliminar" pide con HTMX el aviso de confirmación (no usa la
		// ventana del navegador) y la fila tiene id para que el DELETE la reemplace
		`<li id="fila-5"`,
		`hx-get="/cuenta/suscripciones/5/confirmar-eliminar?mes=2026-10"`,
		`hx-target="#avisos"`,
		`hx-swap="outerHTML"`,
		// el layout carga HTMX
		`<script src="https://unpkg.com/htmx.org@1.9.10"></script>`,
		"Modificar",
		`name="q"`,
		"Gasto mensual (activo): $ 32.000,00",
		"Vence el 28:",
	)
	// la lista no muestra la cuenta ni las fechas (están en el detalle) y el
	// estado se cambia desde "Modificar", no desde la lista
	pruebas.VerificarNoContiene(t, html, "martinagym", `name="estado"`, "Cancelada")

	// orden en cada suscripción: estado, Modificar y Eliminar
	estado := strings.Index(html, `class="estado-fijo"`)
	modificar := strings.Index(html, `class="btn-modificar"`)
	eliminar := strings.Index(html, `class="btn-eliminar"`)
	if !(estado < modificar && modificar < eliminar) {
		t.Errorf("el orden tendría que ser estado, Modificar, Eliminar (posiciones %d, %d, %d)", estado, modificar, eliminar)
	}
}

// la respuesta del DELETE no trae contenido principal (la fila desaparece):
// solo el gasto, el calendario y el aviso, fuera de banda
func TestRespuestaSuscripcionEliminada(t *testing.T) {
	d := DatosSuscripciones{
		Mes: "2026-10", NombreMes: "octubre de 2026", EsMesActual: true,
		HayAlguna: true, HayEnElMes: true, Total: "$ 0,00",
		Calendario: ArmarCalendario("2026-10", nil, pruebas.Fecha(2026, 10, 29)),
	}
	html := pruebas.Renderizar(t, RespuestaSuscripcionEliminada(d, vista.Aviso{Texto: "Suscripción eliminada."}))

	pruebas.VerificarContiene(t, html,
		`<p id="total-mes" class="total" hx-swap-oob="true">`,
		`<aside id="panel-calendario" class="panel calendario" aria-label="Calendario de vencimientos" hx-swap-oob="true">`,
		`<div id="avisos" hx-swap-oob="true">`,
		"Suscripción eliminada.",
	)
	if !strings.HasPrefix(strings.TrimSpace(html), `<p id="total-mes"`) {
		t.Errorf("la respuesta no tendría que tener contenido antes de los fragmentos oob: %q", html[:60])
	}
	pruebas.VerificarNoContiene(t, html, "<li", "<html")
}

// la respuesta del POST: la lista de activas y el formulario vacío fuera de banda
func TestDetalle_ModificarConEstado(t *testing.T) {
	s := pruebas.Fila()
	d := DatosSuscripciones{
		Mes: "2026-10", NombreMes: "octubre de 2026", EsMesActual: true,
		Suscripciones: []vista.Fila{s}, HayEnElMes: true, HayAlguna: true,
		Editables: map[int32]bool{5: true},
		Detalle: &Detalle{Sub: s, Editable: true, Editando: true,
			Form: FormModificar{Monto: "32000", UsuarioCuenta: "martinagym", Estado: "Pausada"}},
	}
	html := pruebas.Renderizar(t, MisSuscripcionesPage(vista.Pagina{Clave: "suscripciones", Sesion: &vista.Sesion{Nombre: "Nico"}}, d))

	pruebas.VerificarContiene(t, html,
		`action="/cuenta/suscripciones/5/modificar"`,
		`<select name="estado">`,
		`<option value="Activa">Activa</option>`,
		`<option value="Pausada" selected>Pausada</option>`,
	)
	// no existe el estado Cancelada: para dejar una suscripción se la elimina
	pruebas.VerificarNoContiene(t, html, "Cancelada")
}

func TestMisSuscripcionesPage_SoloLecturaYDetalle(t *testing.T) {
	s := pruebas.Fila()
	s.Estado = "Vencida"
	password := "gym2026!"
	d := DatosSuscripciones{
		Mes: "2026-09", NombreMes: "septiembre de 2026", SoloLectura: true,
		Suscripciones: []vista.Fila{s}, HayEnElMes: true, HayAlguna: true,
		Editables: map[int32]bool{5: false},
		Detalle:   &Detalle{Sub: s, PasswordRevelada: true, Password: &password},
	}
	html := pruebas.Renderizar(t, MisSuscripcionesPage(vista.Pagina{Clave: "suscripciones", Sesion: &vista.Sesion{Nombre: "Nico"}}, d))

	pruebas.VerificarContiene(t, html,
		"solo de lectura",
		"Vencida (renovada)",
		"martinagym",
		"gym2026!",
		`class="detalle-fondo"`,
	)
	pruebas.VerificarNoContiene(t, html, `/estado"`, "hx-delete", `/modificar"`)
}

func TestArmarCalendario(t *testing.T) {
	celdas := ArmarCalendario("2026-10", []vista.Fila{pruebas.Fila()}, pruebas.Fecha(2026, 10, 29))

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

func TestItemSuscripcion_EscapaHTML(t *testing.T) {
	s := pruebas.Fila()
	s.ServicioNombre = "<script>alert(1)</script>"
	html := pruebas.Renderizar(t, ItemSuscripcion(DatosSuscripciones{Editables: map[int32]bool{}}, s))
	pruebas.VerificarNoContiene(t, html, "<script>")
}

func TestGastoPendiente(t *testing.T) {
	fila := func(estado, monto string, vence time.Time) vista.Fila {
		return vista.Fila{Estado: estado, Monto: monto, FechaVencimiento: vence}
	}
	subs := []vista.Fila{
		fila("Activa", "1000", pruebas.Fecha(2026, 11, 2)),   // ya venció este mes: no se cuenta
		fila("Activa", "2000", pruebas.Fecha(2026, 11, 4)),   // vence hoy: falta pagarla
		fila("Activa", "3000", pruebas.Fecha(2026, 11, 20)),  // falta pagarla
		fila("Pausada", "4000", pruebas.Fecha(2026, 11, 25)), // pausada: no suma
		fila("Vencida", "5000", pruebas.Fecha(2026, 11, 1)),  // período ya renovado (pagado)
	}
	hoy := time.Date(2026, 11, 4, 15, 30, 0, 0, time.Local)

	if obtenido := GastoPendiente(subs, hoy); obtenido != "$ 5.000,00" {
		t.Errorf("GastoPendiente = %s, se esperaba $ 5.000,00 (solo lo que falta pagar)", obtenido)
	}
}

// TP6: la confirmación del borrado es el aviso negro de la página (#avisos),
// no la ventana del navegador (hx-confirm)
func TestConfirmarEliminar(t *testing.T) {
	html := pruebas.Renderizar(t, ConfirmarEliminar("2026-10", "", pruebas.Fila()))

	pruebas.VerificarContiene(t, html,
		`<div id="avisos">`,
		`class="mensaje error mensaje-confirmar"`,
		"¿Seguro que querés eliminar la suscripción a Gimnasio? No se puede deshacer.",
		// "Sí, eliminar": DELETE que reemplaza la fila
		`hx-delete="/cuenta/suscripciones/5?mes=2026-10"`,
		`hx-target="#fila-5"`,
		// "Cancelar": pide el aviso vacío
		`hx-get="/cuenta/suscripciones/5/confirmar-eliminar?cancelar=1&amp;mes=2026-10"`,
	)
	pruebas.VerificarNoContiene(t, html, "hx-confirm")
}
