package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
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

type ManejadorPago struct {
	servicio *ServicioPago
}

func NuevoManejadorPago(s *ServicioPago) *ManejadorPago {
	return &ManejadorPago{servicio: s}
}

// ManejarPagos atiende solicitudes POST en /pagos
func (h *ManejadorPago) ManejarPagos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.crear(w, r)
	default:
		http.Error(w, "Método HTTP no permitido", http.StatusMethodNotAllowed)
	}
}

// ManejarPagoPorID atiende GET y DELETE en /pagos/{id}
func (h *ManejadorPago) ManejarPagoPorID(w http.ResponseWriter, r *http.Request) {
	id, ok := h.extraerID(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		if h.pagoEsDelUsuario(w, r, id) {
			h.obtenerPorID(w, r, id)
		}
	case http.MethodDelete:
		if h.pagoEsDelUsuario(w, r, id) {
			h.eliminar(w, r, id)
		}
	default:
		http.Error(w, "Método HTTP no permitido (los registros de pago son inmutables)", http.StatusMethodNotAllowed)
	}
}

// ManejarPagosPorSuscripcion atiende GET en /pagos/suscripcion/{id_suscripcion}
func (h *ManejadorPago) ManejarPagosPorSuscripcion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método HTTP no permitido", http.StatusMethodNotAllowed)
		return
	}

	idSuscripcion, ok := h.extraerID(w, r)
	if !ok {
		return
	}

	if h.suscripcionEsDelUsuario(w, r, idSuscripcion) {
		h.obtenerPorSuscripcion(w, r, idSuscripcion)
	}
}

// pagoEsDelUsuario y suscripcionEsDelUsuario responden 404 si el registro no
// existe o es de otro usuario, para no revelar datos ajenos.
func (h *ManejadorPago) pagoEsDelUsuario(w http.ResponseWriter, r *http.Request, id int32) bool {
	usr, _ := UsuarioDeContexto(r.Context())
	propio, err := h.servicio.PerteneceA(r.Context(), id, usr.IDUsuario)
	if err != nil {
		h.responderError(w, err)
		return false
	}
	if !propio {
		http.Error(w, "pago no encontrado", http.StatusNotFound)
		return false
	}
	return true
}

func (h *ManejadorPago) suscripcionEsDelUsuario(w http.ResponseWriter, r *http.Request, idSuscripcion int32) bool {
	usr, _ := UsuarioDeContexto(r.Context())
	propia, err := h.servicio.SuscripcionPerteneceA(r.Context(), idSuscripcion, usr.IDUsuario)
	if err != nil {
		h.responderError(w, err)
		return false
	}
	if !propia {
		http.Error(w, "la suscripción especificada no existe", http.StatusNotFound)
		return false
	}
	return true
}

func (h *ManejadorPago) crear(w http.ResponseWriter, r *http.Request) {
	var entrada sqlc.CreatePagoParams
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		http.Error(w, "El formato JSON enviado no es válido", http.StatusBadRequest)
		return
	}

	// si el id es inválido (<= 0) lo rechaza la capa de lógica con 400
	if entrada.IDSuscripcion > 0 && !h.suscripcionEsDelUsuario(w, r, entrada.IDSuscripcion) {
		return
	}

	creado, err := h.servicio.Crear(r.Context(), entrada)
	if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusCreated, creado)
}

func (h *ManejadorPago) obtenerPorID(w http.ResponseWriter, r *http.Request, id int32) {
	pago, err := h.servicio.ObtenerPorID(r.Context(), id)
	if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusOK, pago)
}

func (h *ManejadorPago) obtenerPorSuscripcion(w http.ResponseWriter, r *http.Request, idSuscripcion int32) {
	lista, err := h.servicio.ListarPorSuscripcion(r.Context(), idSuscripcion)
	if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusOK, lista)
}

func (h *ManejadorPago) eliminar(w http.ResponseWriter, r *http.Request, id int32) {
	if err := h.servicio.Eliminar(r.Context(), id); err != nil {
		h.responderError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ManejadorPago) extraerID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	partes := strings.Split(r.URL.Path, "/")
	if len(partes) < 3 || partes[len(partes)-1] == "" {
		http.Error(w, "ID no especificado", http.StatusBadRequest)
		return 0, false
	}

	idNum, err := strconv.Atoi(partes[len(partes)-1])
	if err != nil {
		http.Error(w, "El ID debe ser un valor numérico", http.StatusBadRequest)
		return 0, false
	}
	return int32(idNum), true
}

func (h *ManejadorPago) responderError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no encontrado") || strings.Contains(msg, "no existe"):
		http.Error(w, msg, http.StatusNotFound)
	case strings.Contains(msg, "obligatoria") || strings.Contains(msg, "mayor a cero") || strings.Contains(msg, "posterior") || strings.Contains(msg, "válido"):
		http.Error(w, msg, http.StatusBadRequest)
	default:
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
	}
}

func (h *ManejadorPago) responderJSON(w http.ResponseWriter, estado int, datos any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(datos)
}