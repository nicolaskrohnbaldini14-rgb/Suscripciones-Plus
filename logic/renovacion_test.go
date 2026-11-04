package logic

import (
	"testing"
	"time"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

func fecha(anio int, mes time.Month, dia int) time.Time {
	return time.Date(anio, mes, dia, 0, 0, 0, 0, time.UTC)
}

func TestSiguienteVencimiento(t *testing.T) {
	casos := []struct {
		nombre             string
		inicio, venc, sale time.Time
	}{
		{"un mes", fecha(2026, 10, 28), fecha(2026, 11, 28), fecha(2026, 12, 28)},
		{"un mes cruzando el año", fecha(2026, 11, 15), fecha(2026, 12, 15), fecha(2027, 1, 15)},
		{"tres meses", fecha(2026, 1, 10), fecha(2026, 4, 10), fecha(2026, 7, 10)},
		{"un año", fecha(2025, 1, 1), fecha(2026, 1, 1), fecha(2027, 1, 1)},
		{"plazo en días (no es un mes exacto)", fecha(2026, 10, 1), fecha(2026, 10, 15), fecha(2026, 10, 29)},
	}
	for _, c := range casos {
		if obtenido := siguienteVencimiento(c.inicio, c.venc); !obtenido.Equal(c.sale) {
			t.Errorf("%s: siguienteVencimiento(%s, %s) = %s, se esperaba %s", c.nombre,
				c.inicio.Format("2006-01-02"), c.venc.Format("2006-01-02"),
				obtenido.Format("2006-01-02"), c.sale.Format("2006-01-02"))
		}
	}
}

func TestEsSoloLectura(t *testing.T) {
	// se fija "hoy" en el 29/10/2026
	reloj = func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.Local) }
	t.Cleanup(func() { reloj = time.Now })

	casos := []struct {
		nombre string
		sub    sqlc.Suscripcion
		espera bool
	}{
		{"vence este mes", sqlc.Suscripcion{Estado: "Activa", FechaVencimiento: fecha(2026, 10, 5)}, false},
		{"vence el mes que viene", sqlc.Suscripcion{Estado: "Pausada", FechaVencimiento: fecha(2026, 11, 20)}, false},
		{"venció el mes pasado", sqlc.Suscripcion{Estado: "Pausada", FechaVencimiento: fecha(2026, 9, 30)}, true},
		{"período renovado (Vencida)", sqlc.Suscripcion{Estado: EstadoVencida, FechaVencimiento: fecha(2026, 10, 15)}, true},
	}
	for _, c := range casos {
		if obtenido := esSoloLectura(c.sub); obtenido != c.espera {
			t.Errorf("%s: esSoloLectura = %v, se esperaba %v", c.nombre, obtenido, c.espera)
		}
	}
}

func TestMismaCuenta(t *testing.T) {
	a, b, c := "martinagym", "MartinaGym", "otra"
	if !mismaCuenta(&a, &b) {
		t.Error("las cuentas deberían compararse sin distinguir mayúsculas")
	}
	if mismaCuenta(&a, &c) {
		t.Error("cuentas distintas no deberían ser iguales")
	}
	if !mismaCuenta(nil, nil) || mismaCuenta(&a, nil) {
		t.Error("dos suscripciones sin cuenta cuentan como la misma; con y sin cuenta, no")
	}
}
