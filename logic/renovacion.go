package logic

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// EstadoVencida es el estado de un período que ya venció y se renovó. Solo lo
// asigna la renovación automática: no se puede elegir desde la API.
const EstadoVencida = "Vencida"

// ErrSoloLectura: las suscripciones vencidas o de meses anteriores no se tocan
var ErrSoloLectura = errors.New("las suscripciones vencidas o de meses anteriores no se pueden modificar")

// reloj permite fijar "hoy" en los tests
var reloj = time.Now

// hoy devuelve la fecha actual a las 00:00 UTC, igual que las columnas DATE
func hoy() time.Time {
	ahora := reloj()
	return time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, time.UTC)
}

// siguienteVencimiento calcula el próximo vencimiento manteniendo el mismo
// plazo con el que se registró la suscripción. Si el plazo es una cantidad
// exacta de meses (del 28/10 al 28/11, por ejemplo) se suman meses; si no,
// se suma la misma cantidad de días.
func siguienteVencimiento(inicio, vencimiento time.Time) time.Time {
	meses := (vencimiento.Year()-inicio.Year())*12 + int(vencimiento.Month()) - int(inicio.Month())
	if meses > 0 && inicio.AddDate(0, meses, 0).Equal(vencimiento) {
		return vencimiento.AddDate(0, meses, 0)
	}
	dias := int(vencimiento.Sub(inicio).Hours() / 24)
	return vencimiento.AddDate(0, 0, dias)
}

// EsSoloLectura indica si una suscripción ya no se puede modificar: si fue
// renovada (Vencida) o si vence en un mes anterior al actual. Lo usan la
// lógica (para rechazar cambios) y las vistas (para no ofrecerlos).
func EsSoloLectura(estado string, vencimiento time.Time) bool {
	if estado == EstadoVencida {
		return true
	}
	h := hoy()
	inicioDelMes := time.Date(h.Year(), h.Month(), 1, 0, 0, 0, 0, time.UTC)
	return vencimiento.Before(inicioDelMes)
}

func esSoloLectura(sub sqlc.Suscripcion) bool {
	return EsSoloLectura(sub.Estado, sub.FechaVencimiento)
}

// RenovarVencidas renueva las suscripciones activas cuyo vencimiento ya pasó:
// el período vencido queda como historial (estado Vencida) y se crea el
// siguiente con el mismo monto, cuenta y plazo. Si pasaron varios períodos,
// se renueva las veces que haga falta hasta llegar a uno vigente.
func (s *ServicioSuscripcion) RenovarVencidas(ctx context.Context) (int, error) {
	renovadas := 0
	// el límite evita un ciclo infinito ante un dato inesperado
	for vuelta := 0; vuelta < 1000; vuelta++ {
		pendientes, err := s.consultas.ListSuscripcionesParaRenovar(ctx, hoy())
		if err != nil {
			return renovadas, err
		}
		if len(pendientes) == 0 {
			return renovadas, nil
		}
		for _, sub := range pendientes {
			_, err := s.consultas.RenovarSuscripcion(ctx, sqlc.RenovarSuscripcionParams{
				IDSuscripcion:    sub.IDSuscripcion,
				NuevoVencimiento: siguienteVencimiento(sub.FechaInicio, sub.FechaVencimiento),
			})
			if err == sql.ErrNoRows {
				continue // otra petición la renovó al mismo tiempo
			} else if err != nil {
				return renovadas, err
			}
			renovadas++
		}
	}
	return renovadas, nil
}
