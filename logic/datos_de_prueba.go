package logic

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// UsuarioDePrueba es una cuenta que se crea al iniciar la app para poder
// probarla sin registrarse. Los datos vienen del .env (USUARIO_PRUEBA_1, ...).
type UsuarioDePrueba struct {
	Nombre   string
	Email    string
	Password string
}

// suscripcionDePrueba describe una suscripción de ejemplo. El vencimiento se
// calcula a partir del mes actual (mes 0 = este mes, -1 = el anterior, 1 = el
// siguiente) para que siempre haya vencimientos en el mes que se está viendo.
type suscripcionDePrueba struct {
	servicio       string
	monto          string
	mes            int // mes del vencimiento, relativo al actual
	dia            int // día del vencimiento
	plazoMeses     int // la fecha de inicio es el vencimiento menos este plazo
	usuarioCuenta  string
	passwordCuenta string
	estado         string // "" = queda Activa
}

// ejemplosPorUsuario son las suscripciones que se le cargan a cada usuario de
// prueba (el primero recibe el primer grupo, el segundo el siguiente, etc.).
// Muestran las distintas situaciones de la app: varias en el mes, con y sin
// cuenta, dos cuentas del mismo servicio, una pausada, una cancelada, una del
// mes anterior (que se renueva sola) y otras del mes siguiente.
var ejemplosPorUsuario = [][]suscripcionDePrueba{
	{
		{servicio: "Netflix", monto: "7999", mes: 0, dia: 5, plazoMeses: 1, usuarioCuenta: "martina.netflix", passwordCuenta: "netflix123"},
		{servicio: "Netflix", monto: "5999", mes: 0, dia: 22, plazoMeses: 1, usuarioCuenta: "martina.familia", passwordCuenta: "familia123"},
		{servicio: "Spotify", monto: "4499", mes: 0, dia: 12, plazoMeses: 1, usuarioCuenta: "martina@ejemplo.com", passwordCuenta: "spotify123"},
		{servicio: "Gimnasio", monto: "30000", mes: 0, dia: 18, plazoMeses: 1},
		{servicio: "OSDE", monto: "95000", mes: -1, dia: 15, plazoMeses: 1},
		{servicio: "Microsoft 365", monto: "12000", mes: 1, dia: 10, plazoMeses: 12, usuarioCuenta: "martina@ejemplo.com", estado: "Pausada"},
	},
	{
		{servicio: "Disney+", monto: "6500", mes: 0, dia: 8, plazoMeses: 1, usuarioCuenta: "agustina.disney", passwordCuenta: "disney123"},
		{servicio: "YouTube Premium", monto: "3200", mes: 0, dia: 14, plazoMeses: 1, usuarioCuenta: "agustina@ejemplo.com"},
		{servicio: "Pádel", monto: "18000", mes: 0, dia: 20, plazoMeses: 1},
		{servicio: "Internet hogar", monto: "25000", mes: 0, dia: 25, plazoMeses: 1, usuarioCuenta: "cliente-48213"},
		{servicio: "Seguro de auto", monto: "42000", mes: 1, dia: 3, plazoMeses: 1, estado: "Pausada"},
		{servicio: "HBO Max", monto: "5500", mes: 1, dia: 16, plazoMeses: 1, usuarioCuenta: "agustina.hbo", passwordCuenta: "hbo12345", estado: "Cancelada"},
	},
	{
		{servicio: "HBO Max", monto: "5500", mes: 0, dia: 6, plazoMeses: 1, usuarioCuenta: "nicolas.hbo", passwordCuenta: "hbo12345"},
		{servicio: "Deezer", monto: "3900", mes: 0, dia: 11, plazoMeses: 1, usuarioCuenta: "nicolas@ejemplo.com", passwordCuenta: "deezer123"},
		{servicio: "ChatGPT Plus", monto: "22000", mes: 0, dia: 19, plazoMeses: 1, usuarioCuenta: "nicolas@ejemplo.com"},
		{servicio: "Yoga", monto: "15000", mes: 0, dia: 24, plazoMeses: 1},
		{servicio: "Swiss Medical", monto: "88000", mes: 1, dia: 9, plazoMeses: 1},
	},
}

// CargarUsuariosDePrueba crea los usuarios de prueba que todavía no existen,
// cada uno con sus suscripciones de ejemplo. Usa los mismos servicios que la
// aplicación, así que pasan por las mismas validaciones (y la contraseña se
// guarda hasheada). Devuelve cuántos usuarios creó.
func CargarUsuariosDePrueba(ctx context.Context, q *sqlc.Queries, usuarios []UsuarioDePrueba) (int, error) {
	servicioUsuario := NuevoServicioUsuario(q)
	servicioSuscripcion := NuevoServicioSuscripcion(q)
	creados := 0

	for i, u := range usuarios {
		// si la cuenta ya existe no se toca (no se duplican las suscripciones)
		if _, err := q.GetUsuarioByEmail(ctx, u.Email); err == nil {
			continue
		} else if err != sql.ErrNoRows {
			return creados, err
		}

		usr, err := servicioUsuario.Crear(ctx, sqlc.CreateUsuarioParams{
			Nombre:   u.Nombre,
			Email:    u.Email,
			Password: u.Password,
		})
		if err != nil {
			return creados, fmt.Errorf("usuario de prueba %s: %w", u.Email, err)
		}
		creados++

		for _, ej := range ejemplosPorUsuario[i%len(ejemplosPorUsuario)] {
			if err := crearSuscripcionDePrueba(ctx, q, servicioSuscripcion, usr.IDUsuario, ej); err != nil {
				return creados, fmt.Errorf("suscripción de prueba %s de %s: %w", ej.servicio, u.Email, err)
			}
		}
	}
	return creados, nil
}

func crearSuscripcionDePrueba(ctx context.Context, q *sqlc.Queries, s *ServicioSuscripcion, idUsuario int32, ej suscripcionDePrueba) error {
	srv, err := q.GetServicioByNombre(ctx, ej.servicio)
	if err != nil {
		return err
	}

	h := hoy()
	vencimiento := time.Date(h.Year(), h.Month()+time.Month(ej.mes), ej.dia, 0, 0, 0, 0, time.UTC)
	inicio := vencimiento.AddDate(0, -ej.plazoMeses, 0)

	nueva, err := s.Crear(ctx, sqlc.CreateSubscripcionParams{
		IDUsuario:        idUsuario,
		IDServicio:       srv.IDServicio,
		Monto:            ej.monto,
		FechaInicio:      inicio,
		FechaVencimiento: vencimiento,
		UsuarioCuenta:    textoDePrueba(ej.usuarioCuenta),
		PasswordCuenta:   textoDePrueba(ej.passwordCuenta),
	})
	if err != nil {
		return err
	}

	// una suscripción nueva siempre arranca Activa: pausarla o cancelarla es
	// una modificación posterior, igual que en la app
	if ej.estado == "" {
		return nil
	}
	_, err = s.Actualizar(ctx, nueva.IDSuscripcion, sqlc.UpdateSubscripcionParams{
		Monto:            ej.monto,
		FechaVencimiento: vencimiento,
		Estado:           ej.estado,
		UsuarioCuenta:    textoDePrueba(ej.usuarioCuenta),
	})
	return err
}

func textoDePrueba(valor string) *string {
	if valor == "" {
		return nil
	}
	return &valor
}
