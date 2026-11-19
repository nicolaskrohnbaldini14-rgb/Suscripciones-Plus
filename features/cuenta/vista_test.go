package cuenta

import (
	"testing"

	"tp2-krohn-trapani-cicopiedi/plataforma/pruebas"
	"tp2-krohn-trapani-cicopiedi/plataforma/vista"
)

func TestCuentaPage(t *testing.T) {
	p := vista.Pagina{Clave: "cuenta", Sesion: &vista.Sesion{Nombre: "Nico"}}
	html := pruebas.Renderizar(t, CuentaPage(p,
		ResumenMes{NombreMes: "octubre de 2026", Cantidad: 1, Total: "$ 2.500,00"},
		vista.Activas{Lista: []vista.Fila{pruebas.Fila()}}))

	pruebas.VerificarContiene(t, html,
		"Hola, Nico",
		"En octubre de 2026 vence 1 suscripción · $ 2.500,00",
		`href="/cuenta/suscripciones"`,
		`href="/cuenta/suscripciones/nueva"`,
		// panel con todas las suscripciones activas, con link a su detalle
		"Tus suscripciones activas",
		`<p class="total">1 suscripción</p>`,
		"Gimnasio — $ 32.000,00",
		"Vence el 28/10/2026",
		"martinagym",
		`href="/cuenta/suscripciones?mes=2026-10&amp;ver=5"`,
	)

	// sin activas, el panel lo dice
	vacia := pruebas.Renderizar(t, CuentaPage(p, ResumenMes{NombreMes: "octubre de 2026"}, vista.Activas{}))
	pruebas.VerificarContiene(t, vacia, "No tenés suscripciones activas.")
}
