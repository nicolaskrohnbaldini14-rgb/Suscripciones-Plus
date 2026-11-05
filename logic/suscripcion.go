package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// ErrPasswordUsuario se devuelve cuando, para ver la contraseña de la cuenta
// de un servicio, el usuario no confirma bien su propia contraseña
var ErrPasswordUsuario = fmt.Errorf("la contraseña de tu usuario no es correcta")

// ErrCuentaRequerida y ErrCuentaRepetida aplican cuando un usuario tiene más
// de una suscripción al mismo servicio: cada una tiene que ser otra cuenta.
var (
	ErrCuentaRequerida = errors.New("para agregar otra, indicá el usuario de la cuenta para diferenciarla")
	ErrCuentaRepetida  = errors.New("usá otro usuario de cuenta para diferenciarla")
)

// Estados permitidos según las reglas de negocio. No hay estado "Cancelada":
// para dejar una suscripción se la elimina.
var estadosValidos = map[string]bool{
	"Activa":  true,
	"Pausada": true,
}


// pregunta si esta bien esto o lo cambiamos en el schema
// parsearMonto convierte el monto (string en sqlc) a número y valida que sea > 0
func parsearMonto(monto string) (float64, error) {
	valor, err := strconv.ParseFloat(strings.TrimSpace(monto), 64)
	if err != nil {
		return 0, fmt.Errorf("el monto debe ser un número válido")
	}
	if valor <= 0 {
		return 0, fmt.Errorf("el monto debe ser mayor a cero")
	}
	return valor, nil
}

func validarCamposSuscripcion(datos sqlc.CreateSubscripcionParams) error {
	if datos.IDUsuario <= 0 || datos.IDServicio <= 0 {
		return fmt.Errorf("los identificadores id_usuario e id_servicio deben ser válidos")
	}
	if _, err := parsearMonto(datos.Monto); err != nil {
		return err
	}
	if datos.FechaInicio.IsZero() || datos.FechaVencimiento.IsZero() {
		return fmt.Errorf("las fechas de inicio y vencimiento son obligatorias")
	}
	if !datos.FechaInicio.Before(datos.FechaVencimiento) {
		return fmt.Errorf("la fecha de inicio debe ser anterior a la fecha de vencimiento")
	}

	// una suscripción nueva siempre arranca activa; pausarla o cancelarla
	// es una modificación posterior (PUT)
	est := strings.TrimSpace(datos.Estado)
	if est != "" && !estadosValidos[est] {
		return fmt.Errorf("el estado '%s' no es válido (permitidos: Activa y Pausada)", est)
	}
	if est != "" && est != "Activa" {
		return fmt.Errorf("una suscripción nueva se crea siempre en estado Activa")
	}

	return nil
}

// textoOpcional limpia un texto opcional: si queda vacío se guarda como NULL
func textoOpcional(valor *string) *string {
	if valor == nil {
		return nil
	}
	limpio := strings.TrimSpace(*valor)
	if limpio == "" {
		return nil
	}
	return &limpio
}

// ocultarPasswordCuenta evita devolver la contraseña de la cuenta al cliente
func ocultarPasswordCuenta(sub sqlc.Suscripcion) sqlc.Suscripcion {
	sub.PasswordCuenta = nil
	return sub
}

type ServicioSuscripcion struct {
	consultas *sqlc.Queries
}

func NuevoServicioSuscripcion(q *sqlc.Queries) *ServicioSuscripcion {
	return &ServicioSuscripcion{consultas: q}
}

// Crear valida las FKs, fechas, monto y evita suscripciones duplicadas
func (s *ServicioSuscripcion) Crear(ctx context.Context, entrada sqlc.CreateSubscripcionParams) (sqlc.Suscripcion, error) {
	// Si no viene estado, por defecto la suscripción arranca como 'Activa'
	entrada.Estado = strings.TrimSpace(entrada.Estado)
	if entrada.Estado == "" {
		entrada.Estado = "Activa"
	}

	if err := validarCamposSuscripcion(entrada); err != nil {
		return sqlc.Suscripcion{}, err
	}

	// usuario y contraseña de la cuenta del servicio son opcionales
	entrada.UsuarioCuenta = textoOpcional(entrada.UsuarioCuenta)
	entrada.PasswordCuenta = textoOpcional(entrada.PasswordCuenta)

	// 1. Verificar que el usuario exista
	_, err := s.consultas.GetUsuario(ctx, entrada.IDUsuario)
	if err == sql.ErrNoRows {
		return sqlc.Suscripcion{}, fmt.Errorf("el usuario especificado no existe")
	} else if err != nil {
		return sqlc.Suscripcion{}, err
	}

	// 2. Verificar que el servicio exista
	_, err = s.consultas.GetServicio(ctx, entrada.IDServicio)
	if err == sql.ErrNoRows {
		return sqlc.Suscripcion{}, fmt.Errorf("el servicio especificado no existe")
	} else if err != nil {
		return sqlc.Suscripcion{}, err
	}

	// 3. Se puede tener más de una suscripción al mismo servicio si son cuentas
	// distintas: a partir de la segunda hay que indicar el usuario de cuenta
	if err := s.verificarCuentaDistinta(ctx, entrada.IDUsuario, entrada.IDServicio, entrada.UsuarioCuenta, 0, true); err != nil {
		return sqlc.Suscripcion{}, err
	}

	// el monto ya fue validado en validarCamposSuscripcion
	monto, _ := parsearMonto(entrada.Monto)
	entrada.Monto = fmt.Sprintf("%.2f", monto)

	nueva, err := s.consultas.CreateSubscripcion(ctx, entrada)
	if err != nil {
		return sqlc.Suscripcion{}, err
	}

	return ocultarPasswordCuenta(nueva), nil
}

// ObtenerPorID busca una suscripción por id_suscripcion
func (s *ServicioSuscripcion) ObtenerPorID(ctx context.Context, id int32) (sqlc.Suscripcion, error) {
	sub, err := s.consultas.GetSubscripcion(ctx, id)
	if err == sql.ErrNoRows {
		return sqlc.Suscripcion{}, fmt.Errorf("suscripción no encontrada")
	} else if err != nil {
		return sqlc.Suscripcion{}, err
	}
	return ocultarPasswordCuenta(sub), nil
}

// ListarPorUsuario trae todas las suscripciones registradas para un usuario.
// Antes renueva las que vencieron, así la lista siempre está al día.
func (s *ServicioSuscripcion) ListarPorUsuario(ctx context.Context, idUsuario int32) ([]sqlc.ListSubscripcionesUsuarioRow, error) {
	if _, err := s.RenovarVencidas(ctx); err != nil {
		return nil, err
	}
	return s.consultas.ListSubscripcionesUsuario(ctx, idUsuario)
}

// Actualizar modifica el monto, vencimiento, credenciales o estado de una suscripción
func (s *ServicioSuscripcion) Actualizar(ctx context.Context, id int32, entrada sqlc.UpdateSubscripcionParams) (sqlc.Suscripcion, error) {
	// 0. Buscar la suscripción: si no existe o es de solo lectura (vencida o de
	// un mes anterior) no importa qué datos vengan, no se modifica
	subExistente, err := s.consultas.GetSubscripcion(ctx, id)
	if err == sql.ErrNoRows {
		return sqlc.Suscripcion{}, fmt.Errorf("la suscripción especificada no existe")
	} else if err != nil {
		return sqlc.Suscripcion{}, err
	}
	if esSoloLectura(subExistente) {
		return sqlc.Suscripcion{}, ErrSoloLectura
	}

	// 1. Validaciones básicas de entrada
	monto, err := parsearMonto(entrada.Monto)
	if err != nil {
		return sqlc.Suscripcion{}, err
	}
	if entrada.FechaVencimiento.IsZero() {
		return sqlc.Suscripcion{}, fmt.Errorf("la fecha de vencimiento es obligatoria")
	}
	// 2. Validar que el estado sea uno de los permitidos
	est := strings.TrimSpace(entrada.Estado)
	if !estadosValidos[est] {
		return sqlc.Suscripcion{}, fmt.Errorf("el estado '%s' no es válido (permitidos: Activa y Pausada)", est)
	}

	// 3. Validar coherencia de fechas (la fecha de inicio debe ser anterior al vencimiento)
	if !subExistente.FechaInicio.Before(entrada.FechaVencimiento) {
		return sqlc.Suscripcion{}, fmt.Errorf("la fecha de inicio debe ser anterior a la fecha de vencimiento")
	}

	// las credenciales son opcionales; como la contraseña nunca se devuelve,
	// si no se envía una nueva se conserva la que estaba guardada
	entrada.UsuarioCuenta = textoOpcional(entrada.UsuarioCuenta)
	entrada.PasswordCuenta = textoOpcional(entrada.PasswordCuenta)
	if entrada.PasswordCuenta == nil {
		entrada.PasswordCuenta = subExistente.PasswordCuenta
	}

	// 4. Activa o pausada sigue vigente: no puede repetir la cuenta de otra
	// suscripción vigente al mismo servicio
	if err := s.verificarCuentaDistinta(ctx, subExistente.IDUsuario, subExistente.IDServicio, entrada.UsuarioCuenta, id, false); err != nil {
		return sqlc.Suscripcion{}, err
	}

	// 5. Formatear monto y completar los datos que no vienen del body
	entrada.IDSuscripcion = id // el id siempre viene de la URL
	entrada.Monto = fmt.Sprintf("%.2f", monto)
	entrada.Estado = est

	// 6. Actualizar en la base de datos vía sqlc
	if err := s.consultas.UpdateSubscripcion(ctx, entrada); err != nil {
		return sqlc.Suscripcion{}, err
	}

	// 7. Construir la respuesta (sin password_cuenta)
	return sqlc.Suscripcion{
		IDSuscripcion:    id,
		IDUsuario:        subExistente.IDUsuario,
		IDServicio:       subExistente.IDServicio,
		Monto:            entrada.Monto,
		FechaInicio:      subExistente.FechaInicio,
		FechaVencimiento: entrada.FechaVencimiento,
		UsuarioCuenta:    entrada.UsuarioCuenta,
		Estado:           est,
		CreatedAt:        subExistente.CreatedAt,
	}, nil
}

// RevelarPasswordCuenta devuelve la contraseña guardada de la cuenta del
// servicio (o nil si no tiene) solo si la suscripción es del usuario y este
// confirma su propia contraseña de Suscripciones+.
func (s *ServicioSuscripcion) RevelarPasswordCuenta(ctx context.Context, id, idUsuario int32, passwordUsuario string) (*string, error) {
	usr, err := s.consultas.GetUsuario(ctx, idUsuario)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(passwordUsuario)) != nil {
		return nil, ErrPasswordUsuario
	}

	sub, err := s.consultas.GetSubscripcion(ctx, id)
	if err == sql.ErrNoRows || (err == nil && sub.IDUsuario != idUsuario) {
		return nil, fmt.Errorf("suscripción no encontrada")
	} else if err != nil {
		return nil, err
	}
	return sub.PasswordCuenta, nil
}

// PerteneceA indica si la suscripción existe y es del usuario indicado
func (s *ServicioSuscripcion) PerteneceA(ctx context.Context, id, idUsuario int32) (bool, error) {
	sub, err := s.consultas.GetSubscripcion(ctx, id)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return sub.IDUsuario == idUsuario, nil
}

// Eliminar quita la suscripción previa comprobación
func (s *ServicioSuscripcion) Eliminar(ctx context.Context, id int32) error {
	sub, err := s.consultas.GetSubscripcion(ctx, id)
	if err == sql.ErrNoRows {
		return fmt.Errorf("suscripción no encontrada")
	} else if err != nil {
		return err
	}
	if esSoloLectura(sub) {
		return ErrSoloLectura
	}

	return s.consultas.DeleteSubscripcion(ctx, id)
}

// ---------------------------------------------------------------------------
// Varias suscripciones al mismo servicio: tienen que ser cuentas distintas
// ---------------------------------------------------------------------------

// mismaCuenta compara usuarios de cuenta sin distinguir mayúsculas; dos
// suscripciones sin cuenta (nil) cuentan como la misma
func mismaCuenta(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return strings.EqualFold(*a, *b)
}

// verificarCuentaDistinta controla las suscripciones vigentes (activas o
// pausadas) del usuario al mismo servicio, sin contar la que se está
// modificando (excluir). Al crear (exigirCuenta) la segunda en adelante tiene
// que indicar un usuario de cuenta; en todos los casos no se puede repetir.
func (s *ServicioSuscripcion) verificarCuentaDistinta(ctx context.Context, idUsuario, idServicio int32, cuenta *string, excluir int32, exigirCuenta bool) error {
	vigentes, err := s.consultas.ListSuscripcionesVigentes(ctx, sqlc.ListSuscripcionesVigentesParams{
		IDUsuario:  idUsuario,
		IDServicio: idServicio,
	})
	if err != nil {
		return err
	}

	var otras []sqlc.Suscripcion
	for _, v := range vigentes {
		if v.IDSuscripcion != excluir {
			otras = append(otras, v)
		}
	}
	if len(otras) == 0 {
		return nil
	}

	nombre := "este servicio"
	if srv, err := s.consultas.GetServicio(ctx, idServicio); err == nil {
		nombre = srv.Nombre
	}

	if exigirCuenta && cuenta == nil {
		return fmt.Errorf("ya tenés una suscripción a %s: %w", nombre, ErrCuentaRequerida)
	}
	for _, o := range otras {
		if mismaCuenta(o.UsuarioCuenta, cuenta) {
			if cuenta == nil {
				return fmt.Errorf("ya tenés una suscripción a %s sin usuario de cuenta: %w", nombre, ErrCuentaRepetida)
			}
			return fmt.Errorf("ya tenés una suscripción a %s con la cuenta %q: %w", nombre, *cuenta, ErrCuentaRepetida)
		}
	}
	return nil
}

type ManejadorSuscripcion struct {
	servicio *ServicioSuscripcion
}

func NuevoManejadorSuscripcion(s *ServicioSuscripcion) *ManejadorSuscripcion {
	return &ManejadorSuscripcion{servicio: s}
}

// ManejarSuscripciones atiende POST a /suscripciones
func (h *ManejadorSuscripcion) ManejarSuscripciones(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.crear(w, r)
	default:
		http.Error(w, "Método HTTP no permitido", http.StatusMethodNotAllowed)
	}
}

// ManejarSuscripcionPorID atiende GET, PUT y DELETE en /suscripciones/{id}
func (h *ManejadorSuscripcion) ManejarSuscripcionPorID(w http.ResponseWriter, r *http.Request) {
	// POST /suscripciones/{id}/password muestra la contraseña de la cuenta
	if strings.HasSuffix(r.URL.Path, "/password") {
		h.revelarPassword(w, r)
		return
	}

	id, ok := h.extraerID(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		if h.esDelUsuario(w, r, id) {
			h.obtenerPorID(w, r, id)
		}
	case http.MethodPut:
		if h.esDelUsuario(w, r, id) {
			h.actualizar(w, r, id)
		}
	case http.MethodDelete:
		if h.esDelUsuario(w, r, id) {
			h.eliminar(w, r, id)
		}
	default:
		http.Error(w, "Método HTTP no permitido", http.StatusMethodNotAllowed)
	}
}

type confirmacionPassword struct {
	Password string `json:"password"`
}

type respuestaPassword struct {
	PasswordCuenta *string `json:"password_cuenta"`
}

// revelarPassword atiende POST /suscripciones/{id}/password. El cuerpo trae la
// contraseña del usuario de Suscripciones+ para confirmar que es él.
func (h *ManejadorSuscripcion) revelarPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método HTTP no permitido", http.StatusMethodNotAllowed)
		return
	}

	partes := strings.Split(strings.TrimSuffix(r.URL.Path, "/password"), "/")
	idNum, err := strconv.Atoi(partes[len(partes)-1])
	if err != nil {
		http.Error(w, "El ID debe ser un valor numérico", http.StatusBadRequest)
		return
	}

	var entrada confirmacionPassword
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		http.Error(w, "El formato JSON enviado no es válido", http.StatusBadRequest)
		return
	}

	usr, _ := UsuarioDeContexto(r.Context())
	password, err := h.servicio.RevelarPasswordCuenta(r.Context(), int32(idNum), usr.IDUsuario, entrada.Password)
	if err == ErrPasswordUsuario {
		// 403 y no 401: la sesión es válida, lo que falla es la confirmación
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	} else if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusOK, respuestaPassword{PasswordCuenta: password})
}

// esDelUsuario verifica que la suscripción sea del usuario de la sesión.
// Si es de otro usuario responde 404, como si no existiera.
func (h *ManejadorSuscripcion) esDelUsuario(w http.ResponseWriter, r *http.Request, id int32) bool {
	usr, _ := UsuarioDeContexto(r.Context())
	propia, err := h.servicio.PerteneceA(r.Context(), id, usr.IDUsuario)
	if err != nil {
		h.responderError(w, err)
		return false
	}
	if !propia {
		http.Error(w, "suscripción no encontrada", http.StatusNotFound)
		return false
	}
	return true
}

// ManejarSuscripcionesPorUsuario atiende GET en /suscripciones/usuario/{id_usuario}
func (h *ManejadorSuscripcion) ManejarSuscripcionesPorUsuario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método HTTP no permitido", http.StatusMethodNotAllowed)
		return
	}

	idUsuario, ok := h.extraerID(w, r)
	if !ok {
		return
	}

	if usr, _ := UsuarioDeContexto(r.Context()); usr.IDUsuario != idUsuario {
		http.Error(w, "Solo podés ver tus propias suscripciones", http.StatusForbidden)
		return
	}

	h.obtenerPorUsuario(w, r, idUsuario)
}

func (h *ManejadorSuscripcion) crear(w http.ResponseWriter, r *http.Request) {
	var entrada sqlc.CreateSubscripcionParams
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		http.Error(w, "El formato JSON enviado no es válido", http.StatusBadRequest)
		return
	}

	// la suscripción se crea a nombre del usuario de la sesión
	usr, _ := UsuarioDeContexto(r.Context())
	if entrada.IDUsuario == 0 {
		entrada.IDUsuario = usr.IDUsuario
	} else if entrada.IDUsuario != usr.IDUsuario {
		http.Error(w, "Solo podés crear suscripciones a tu nombre", http.StatusForbidden)
		return
	}

	creada, err := h.servicio.Crear(r.Context(), entrada)
	if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusCreated, creada)
}

func (h *ManejadorSuscripcion) obtenerPorID(w http.ResponseWriter, r *http.Request, id int32) {
	sub, err := h.servicio.ObtenerPorID(r.Context(), id)
	if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusOK, sub)
}

func (h *ManejadorSuscripcion) actualizar(w http.ResponseWriter, r *http.Request, id int32) {
	var entrada sqlc.UpdateSubscripcionParams
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		http.Error(w, "El formato JSON enviado no es válido", http.StatusBadRequest)
		return
	}

	actualizada, err := h.servicio.Actualizar(r.Context(), id, entrada)
	if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusOK, actualizada)
}

func (h *ManejadorSuscripcion) eliminar(w http.ResponseWriter, r *http.Request, id int32) {
	if err := h.servicio.Eliminar(r.Context(), id); err != nil {
		h.responderError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ManejadorSuscripcion) obtenerPorUsuario(w http.ResponseWriter, r *http.Request, idUsuario int32) {
	lista, err := h.servicio.ListarPorUsuario(r.Context(), idUsuario)
	if err != nil {
		h.responderError(w, err)
		return
	}
	h.responderJSON(w, http.StatusOK, lista)
}

func (h *ManejadorSuscripcion) extraerID(w http.ResponseWriter, r *http.Request) (int32, bool) {
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

func (h *ManejadorSuscripcion) responderError(w http.ResponseWriter, err error) {
	msg := err.Error()

	switch {
	case errors.Is(err, ErrSoloLectura):
		http.Error(w, msg, http.StatusForbidden)
	case errors.Is(err, ErrCuentaRequerida):
		http.Error(w, msg, http.StatusBadRequest)
	case errors.Is(err, ErrCuentaRepetida):
		http.Error(w, msg, http.StatusConflict)
	case strings.Contains(msg, "no encontrada") || strings.Contains(msg, "no existe"):
		http.Error(w, msg, http.StatusNotFound)
	case strings.Contains(msg, "obligatoria") || strings.Contains(msg, "mayor a cero") || strings.Contains(msg, "anterior") || strings.Contains(msg, "válido") || strings.Contains(msg, "siempre en estado"):
		http.Error(w, msg, http.StatusBadRequest)
	case strings.Contains(msg, "activa"):
		http.Error(w, msg, http.StatusConflict)
	default:
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
	}
}

func (h *ManejadorSuscripcion) responderJSON(w http.ResponseWriter, estado int, datos any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(datos)
}