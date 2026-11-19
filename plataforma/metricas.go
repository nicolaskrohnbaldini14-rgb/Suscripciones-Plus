package plataforma

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Monitoreo con Prometheus (clase de monitoreo). La app expone sus métricas
// en GET /metrics en el formato de texto de Prometheus, y el servidor de
// Prometheus (servicio "prometheus" del docker-compose) las va a buscar cada
// pocos segundos. Se miden las tres cosas del marco RED: cuántos pedidos
// llegan (Rate), cuántos fallan (Errors, por el código de respuesta) y cuánto
// tardan (Duration). Además del cliente salen gratis las métricas go_* y
// process_* (goroutines, memoria, CPU).
//
// promauto registra cada métrica al crearla (equivale a MustRegister).
var (
	pedidosHTTP = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Pedidos HTTP atendidos, por método, ruta y código de respuesta.",
		},
		[]string{"method", "path", "status"},
	)

	duracionHTTP = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duración de los pedidos HTTP en segundos, por método y ruta.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// métricas del negocio: lo que hacen los usuarios en la app

	// SuscripcionesCreadas cuenta las suscripciones creadas desde la app
	SuscripcionesCreadas = promauto.NewCounter(prometheus.CounterOpts{
		Name: "suscripciones_creadas_total",
		Help: "Suscripciones creadas por los usuarios.",
	})

	// SuscripcionesEliminadas cuenta las suscripciones eliminadas desde la app
	SuscripcionesEliminadas = promauto.NewCounter(prometheus.CounterOpts{
		Name: "suscripciones_eliminadas_total",
		Help: "Suscripciones eliminadas por los usuarios.",
	})

	// IniciosDeSesion cuenta los intentos de login, según el resultado
	// ("ok" o "error": email o contraseña incorrectos)
	IniciosDeSesion = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "inicios_de_sesion_total",
		Help: "Intentos de inicio de sesión, por resultado (ok o error).",
	}, []string{"resultado"})
)

// respuestaConEstado guarda el código de respuesta para poder medirlo (si el
// handler no llama a WriteHeader, el código es 200)
type respuestaConEstado struct {
	http.ResponseWriter
	estado int
}

func (r *respuestaConEstado) WriteHeader(estado int) {
	r.estado = estado
	r.ResponseWriter.WriteHeader(estado)
}

// Metricas envuelve todas las rutas y mide cada pedido una sola vez (en vez
// de instrumentar cada handler a mano). Como label "path" usa el patrón de la
// ruta que atendió el pedido (por ejemplo "DELETE /cuenta/suscripciones/{id}")
// y no la URL real con el id: así hay pocas combinaciones de labels (baja
// cardinalidad), como recomienda la clase.
func Metricas(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		respuesta := &respuestaConEstado{ResponseWriter: w, estado: http.StatusOK}

		siguiente.ServeHTTP(respuesta, r)

		// el ServeMux completa r.Pattern al elegir la ruta; si ninguna
		// coincidió (404), no se usa la URL para no crear una serie por URL
		ruta := r.Pattern
		if ruta == "" {
			ruta = "sin ruta"
		}
		pedidosHTTP.WithLabelValues(r.Method, ruta, strconv.Itoa(respuesta.estado)).Inc()
		duracionHTTP.WithLabelValues(r.Method, ruta).Observe(time.Since(inicio).Seconds())
	})
}

// HandlerMetricas atiende GET /metrics con las métricas registradas
func HandlerMetricas() http.Handler {
	return promhttp.Handler()
}
