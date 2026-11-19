package vista

import (
	"testing"
	"time"
)

func TestMeses(t *testing.T) {
	if NombreMes("2026-10") != "octubre de 2026" {
		t.Errorf("NombreMes = %q", NombreMes("2026-10"))
	}
	if SumarMeses("2026-12", 1) != "2027-01" || SumarMeses("2026-01", -1) != "2025-12" {
		t.Error("SumarMeses no cruza bien los años")
	}
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
		if obtenido := FormatearNumero(valor); obtenido != esperado {
			t.Errorf("FormatearNumero(%v) = %q, se esperaba %q", valor, obtenido, esperado)
		}
	}
}

// en un mes anterior se cuenta todo lo que venció (activas y renovadas)
func TestTotalActivo(t *testing.T) {
	fila := func(estado, monto string) Fila {
		return Fila{Estado: estado, Monto: monto, FechaVencimiento: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)}
	}
	subs := []Fila{fila("Activa", "1000"), fila("Activa", "5000"), fila("Pausada", "4000"), fila("Vencida", "5000")}
	if obtenido := TotalActivo(subs); obtenido != "$ 11.000,00" {
		t.Errorf("TotalActivo = %s, se esperaba $ 11.000,00", obtenido)
	}
}
