package logic

import "testing"

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
