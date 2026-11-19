package plataforma

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	nombreCookie   = "sesion"
	duracionSesion = 24 * time.Hour
)

// Sesiones guarda en memoria qué usuario corresponde a cada token de sesión.
// El navegador solo recibe el token aleatorio en una cookie HttpOnly.
// Al reiniciar el servidor las sesiones se pierden y hay que volver a ingresar.
type Sesiones struct {
	mu    sync.RWMutex
	datos map[string]sesion
}

type sesion struct {
	idUsuario int32
	vence     time.Time
}

func NuevasSesiones() *Sesiones {
	return &Sesiones{datos: make(map[string]sesion)}
}

// Iniciar crea una sesión para el usuario y envía la cookie
func (s *Sesiones) Iniciar(w http.ResponseWriter, idUsuario int32) error {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}
	token := hex.EncodeToString(bytes)
	vence := time.Now().Add(duracionSesion)

	s.mu.Lock()
	s.datos[token] = sesion{idUsuario: idUsuario, vence: vence}
	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     nombreCookie,
		Value:    token,
		Path:     "/",
		Expires:  vence,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// Usuario devuelve el id del usuario de la sesión actual, si hay una válida
func (s *Sesiones) Usuario(r *http.Request) (int32, bool) {
	cookie, err := r.Cookie(nombreCookie)
	if err != nil {
		return 0, false
	}

	s.mu.RLock()
	ses, ok := s.datos[cookie.Value]
	s.mu.RUnlock()

	if !ok || time.Now().After(ses.vence) {
		return 0, false
	}
	return ses.idUsuario, true
}

// Cerrar elimina la sesión y le indica al navegador que borre la cookie
func (s *Sesiones) Cerrar(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(nombreCookie); err == nil {
		s.mu.Lock()
		delete(s.datos, cookie.Value)
		s.mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     nombreCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
