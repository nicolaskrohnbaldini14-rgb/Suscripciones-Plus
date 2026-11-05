package logic

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// parsearMontoPago convierte el monto (string en sqlc) a número y valida que sea > 0
func parsearMontoPago(monto string) (float64, error) {
	valor, err := strconv.ParseFloat(strings.TrimSpace(monto), 64)
	if err != nil {
		return 0, fmt.Errorf("el monto del pago debe ser un número válido")
	}
	if valor <= 0 {
		return 0, fmt.Errorf("el monto del pago debe ser mayor a cero")
	}
	// tiene que entrar en la columna monto (DECIMAL(10,2)): si no, la base
	// lo rechaza y la app respondería "Error interno del servidor"
	if valor > MontoMaximo {
		return 0, fmt.Errorf("el monto del pago debe ser un número válido de hasta $ 99.999.999,99")
	}
	return valor, nil
}

func validarCamposPago(datos sqlc.CreatePagoParams) error {
	if datos.IDSuscripcion <= 0 {
		return fmt.Errorf("el id_suscripcion debe ser un identificador válido")
	}
	if _, err := parsearMontoPago(datos.Monto); err != nil {
		return err
	}
	if datos.FechaPago.IsZero() {
		return fmt.Errorf("la fecha de pago es obligatoria")
	}
	return nil
}

type ServicioPago struct {
	consultas *sqlc.Queries
}

func NuevoServicioPago(q *sqlc.Queries) *ServicioPago {
	return &ServicioPago{consultas: q}
}

// Crear valida los datos, la existencia de la suscripción y el rango de la fecha de pago
func (s *ServicioPago) Crear(ctx context.Context, entrada sqlc.CreatePagoParams) (sqlc.Pago, error) {
	if err := validarCamposPago(entrada); err != nil {
		return sqlc.Pago{}, err
	}

	// 1. Verificar que la suscripción exista en la base de datos
	sub, err := s.consultas.GetSubscripcion(ctx, entrada.IDSuscripcion)
	if err == sql.ErrNoRows {
		return sqlc.Pago{}, fmt.Errorf("la suscripción especificada no existe")
	} else if err != nil {
		return sqlc.Pago{}, err
	}

	// La fecha de pago debe estar en el rango de la fecha_vencimiento
	if entrada.FechaPago.After(sub.FechaVencimiento) {
		return sqlc.Pago{}, fmt.Errorf("la fecha del pago no puede ser posterior a la fecha de vencimiento de la suscripción (%s)", sub.FechaVencimiento.Format("2006-01-02"))
	}

	// el monto ya fue validado en validarCamposPago
	monto, _ := parsearMontoPago(entrada.Monto)
	entrada.Monto = fmt.Sprintf("%.2f", monto)

	return s.consultas.CreatePago(ctx, entrada)
}

// ObtenerPorID busca un pago por su identificador único id_pago
func (s *ServicioPago) ObtenerPorID(ctx context.Context, id int32) (sqlc.Pago, error) {
	pago, err := s.consultas.GetPago(ctx, id)
	if err == sql.ErrNoRows {
		return sqlc.Pago{}, fmt.Errorf("pago no encontrado")
	} else if err != nil {
		return sqlc.Pago{}, err
	}
	return pago, nil
}

// ListarPorSuscripcion recupera el historial de pagos de una suscripción
func (s *ServicioPago) ListarPorSuscripcion(ctx context.Context, idSuscripcion int32) ([]sqlc.Pago, error) {
	// Verificar previamente que la suscripción exista
	_, err := s.consultas.GetSubscripcion(ctx, idSuscripcion)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("la suscripción especificada no existe")
	} else if err != nil {
		return nil, err
	}

	return s.consultas.ListPagos(ctx, idSuscripcion)
}

// Eliminar quita un registro de pago previa comprobación
func (s *ServicioPago) Eliminar(ctx context.Context, id int32) error {
	_, err := s.consultas.GetPago(ctx, id)
	if err == sql.ErrNoRows {
		return fmt.Errorf("pago no encontrado")
	} else if err != nil {
		return err
	}

	return s.consultas.DeletePago(ctx, id)
}

// SuscripcionPerteneceA indica si la suscripción existe y es del usuario indicado
func (s *ServicioPago) SuscripcionPerteneceA(ctx context.Context, idSuscripcion, idUsuario int32) (bool, error) {
	sub, err := s.consultas.GetSubscripcion(ctx, idSuscripcion)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return sub.IDUsuario == idUsuario, nil
}

// PerteneceA indica si el pago existe y corresponde a una suscripción del usuario
func (s *ServicioPago) PerteneceA(ctx context.Context, id, idUsuario int32) (bool, error) {
	pago, err := s.consultas.GetPago(ctx, id)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return s.SuscripcionPerteneceA(ctx, pago.IDSuscripcion, idUsuario)
}
