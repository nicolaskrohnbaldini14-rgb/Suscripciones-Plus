package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Health check de la aplicación: dos rutas para saber cómo está la instancia
// (por ejemplo, para que el makefile espere a que esté lista antes de probarla).
//   - GET /salud/vivo (liveness): el proceso está corriendo y atiende pedidos.
//   - GET /salud/lista (readiness): además puede trabajar, porque la base de
//     datos responde. Si no responde, devuelve 503 Service Unavailable.
func registrarSalud(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/salud/vivo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Método HTTP no permitido", http.StatusMethodNotAllowed)
			return
		}
		responderSalud(w, http.StatusOK, "vivo")
	})

	mux.HandleFunc("/salud/lista", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Método HTTP no permitido", http.StatusMethodNotAllowed)
			return
		}
		// si la base tarda más de 2 segundos en contestar, se considera caída
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			log.Printf("Health check: la base de datos no responde: %v", err)
			responderSalud(w, http.StatusServiceUnavailable, "sin base de datos")
			return
		}
		responderSalud(w, http.StatusOK, "lista")
	})
}

func responderSalud(w http.ResponseWriter, codigo int, estado string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(codigo)
	_ = json.NewEncoder(w).Encode(map[string]string{"estado": estado})
}
