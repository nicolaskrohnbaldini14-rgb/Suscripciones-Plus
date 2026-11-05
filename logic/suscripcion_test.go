package logic

import (
	"errors"
	"testing"
)

// cuentas distintas del mismo servicio (verificarCuentaDistinta usa mismaCuenta)
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

func TestParsearMonto(t *testing.T) {
	validos := map[string]float64{"4500": 4500, " 15.50 ": 15.5, "0.01": 0.01, "99999999.99": 99999999.99}
	for texto, espera := range validos {
		if valor, err := parsearMonto(texto); err != nil || valor != espera {
			t.Errorf("parsearMonto(%q) = %v, %v; se esperaba %v", texto, valor, err, espera)
		}
	}
	// no es un número, no es mayor a cero o no entra en DECIMAL(10,2)
	for _, texto := range []string{"", "abc", "0", "-5", "100000000", "1e20", "NaN", "Inf"} {
		if _, err := parsearMonto(texto); !errors.Is(err, ErrMontoInvalido) {
			t.Errorf("parsearMonto(%q) debería dar ErrMontoInvalido, dio %v", texto, err)
		}
	}
}
