package nuevasuscripcion

import (
	"strings"
	"testing"

	"tp2-krohn-trapani-cicopiedi/plataforma/pruebas"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

// la respuesta del POST: la lista de activas y el formulario vacío fuera de banda
func TestRespuestaSuscripcionCreada(t *testing.T) {
	activas := vista.Activas{Lista: []vista.Fila{pruebas.Fila()}}
	html := pruebas.Renderizar(t, RespuestaSuscripcionCreada(pruebas.Catalogo, activas, vista.Aviso{Texto: "Suscripción creada."}))

	pruebas.VerificarContiene(t, html,
		`<section id="panel-activas"`,
		"Gimnasio — $ 32.000,00",
		`id="form-suscripcion"`,
		`hx-swap-oob="true"`,
		`hx-post="/cuenta/suscripciones"`,
		`hx-target="#panel-activas"`,
		"Suscripción creada.",
	)
	if !strings.HasPrefix(html, `<section id="panel-activas"`) {
		t.Errorf("la respuesta tiene que empezar con la lista: %q", html[:60])
	}
	// formulario vacío: sin servicio elegido ni monto
	pruebas.VerificarContiene(t, html, `<input type="hidden" name="id_servicio" value="0">`, `name="monto" value=""`)
	pruebas.VerificarNoContiene(t, html, "<html")
}

func TestNuevaSuscripcionPage(t *testing.T) {
	p := vista.Pagina{Clave: "nueva-suscripcion", Sesion: &vista.Sesion{Nombre: "Nico"}, Servicios: pruebas.Catalogo,
		Catalogo: vista.Catalogo{Abierto: true, Elegir: true, IDActual: 3, CerrarURL: "/cuenta/suscripciones/nueva?servicio=3"}}
	html := pruebas.Renderizar(t, NuevaSuscripcionPage(p, FormSuscripcion{IDServicio: 3, Monto: "15000"}, vista.Activas{}))

	pruebas.VerificarContiene(t, html,
		// el servicio va oculto y el botón muestra el elegido y abre el catálogo
		`<input type="hidden" name="id_servicio" value="3">`,
		`formaction="/cuenta/suscripciones/nueva/catalogo"`,
		"Pádel",
		`value="15000"`,
		// el catálogo abierto en modo "elegir" con el servicio actual resaltado
		"panel-servicios abierto modo-elegir",
		"panel-servicio elegido",
	)
	// TP6: el formulario se envía con HTMX y la respuesta reemplaza a la lista
	pruebas.VerificarContiene(t, html,
		`hx-post="/cuenta/suscripciones"`,
		`hx-target="#panel-activas"`,
		`hx-swap="outerHTML"`,
		`<section id="panel-activas"`,
	)
	// ya no hay lista desplegable del navegador
	pruebas.VerificarNoContiene(t, html, `<select name="id_servicio"`, "<optgroup")

	// abierto desde el formulario: cada servicio es un botón que reenvía el
	// formulario (para no perder lo escrito)
	p.Catalogo.Formulario = IDFormularioSuscripcion
	conFormulario := pruebas.Renderizar(t, NuevaSuscripcionPage(p, FormSuscripcion{IDServicio: 3, Monto: "15000"}, vista.Activas{}))
	pruebas.VerificarContiene(t, conFormulario,
		`form="form-suscripcion"`,
		`name="elegir_servicio" value="3"`,
	)
	// se crea siempre activa: no se elige el estado ni el usuario
	pruebas.VerificarNoContiene(t, html, `name="estado"`, `name="id_usuario"`)
}
