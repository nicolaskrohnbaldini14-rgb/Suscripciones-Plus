package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // driver de PostgreSQL
	"github.com/joho/godotenv"

	"tp2-krohn-trapani-cicopiedi/db/sqlc"
	"tp2-krohn-trapani-cicopiedi/handlers"
	"tp2-krohn-trapani-cicopiedi/logic"
)

func main() {
	// 1. Conexión a PostgreSQL (Docker). El esquema lo crea el contenedor al iniciar
	//    (db/schema/schema.sql se monta en docker-entrypoint-initdb.d).
	//    Las credenciales salen del .env
	connStr, err := cadenaConexion()
	if err != nil {
		log.Fatal(err)
	}

	dbConn, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Error al preparar el pool de conexiones: %v", err)
	}
	defer dbConn.Close()

	if err := esperarBaseDeDatos(dbConn, esperaBaseDeDatos); err != nil {
		log.Fatalf("No se pudo conectar a PostgreSQL: %v", err)
	}
	log.Println("Conexión exitosa a PostgreSQL")

	// 2. Capa de datos (sqlc)
	queries := sqlc.New(dbConn)

	// 3. Catálogo de servicios precargado (no duplica si ya existe)
	ctx := context.Background()
	if err := logic.NuevoServicioServicio(queries).CargarCatalogoInicial(ctx); err != nil {
		log.Fatalf("Error al precargar el catálogo de servicios: %v", err)
	}
	log.Println("Catálogo de servicios precargado")

	// usuarios de prueba del .env, con suscripciones de ejemplo
	cargarUsuariosDePrueba(queries)

	// las suscripciones activas que ya vencieron se renuevan por el mismo plazo
	if renovadas, err := logic.NuevoServicioSuscripcion(queries).RenovarVencidas(ctx); err != nil {
		log.Fatalf("Error al renovar suscripciones vencidas: %v", err)
	} else if renovadas > 0 {
		log.Printf("Suscripciones renovadas: %d", renovadas)
	}

	// 4. Handlers que renderizan las vistas templ (la lógica de negocio sigue en logic/)
	mux := http.NewServeMux()
	handlers.NuevoWeb(queries, handlers.NuevasSesiones()).Registrar(mux)

	// 5. Servidor HTTP
	log.Println("Servidor escuchando en http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Error al iniciar el servidor HTTP: %v", err)
	}
}

// esperaBaseDeDatos es cuánto se reintenta conectar antes de rendirse: si la
// base tarda en arrancar, la aplicación espera en lugar de fallar enseguida
const esperaBaseDeDatos = 60 * time.Second

// cadenaConexion arma la URL de PostgreSQL con las variables de entorno. Si
// existe un archivo .env, primero lo carga (las variables ya definidas en el
// sistema tienen prioridad). Así las credenciales no quedan escritas en el
// código.
func cadenaConexion() (string, error) {
	_ = godotenv.Load() // si no hay .env se usan las variables del sistema

	valores := map[string]string{}
	for _, nombre := range []string{"POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB", "POSTGRES_HOST", "POSTGRES_PORT"} {
		valor := os.Getenv(nombre)
		if valor == "" {
			return "", fmt.Errorf("falta la variable de entorno %s: copiá .env.example como .env", nombre)
		}
		valores[nombre] = valor
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(valores["POSTGRES_USER"], valores["POSTGRES_PASSWORD"]),
		Host:     valores["POSTGRES_HOST"] + ":" + valores["POSTGRES_PORT"],
		Path:     valores["POSTGRES_DB"],
		RawQuery: "sslmode=disable",
	}
	return u.String(), nil
}

// esperarBaseDeDatos reintenta el Ping cada un segundo hasta que la base
// responda o se cumpla el tiempo máximo
func esperarBaseDeDatos(db *sql.DB, maximo time.Duration) error {
	limite := time.Now().Add(maximo)
	for {
		err := db.Ping()
		if err == nil {
			return nil
		}
		if time.Now().After(limite) {
			return fmt.Errorf("la base no respondió en %s: %w", maximo, err)
		}
		log.Printf("Esperando a PostgreSQL... (%v)", err)
		time.Sleep(time.Second)
	}
}

// cargarUsuariosDePrueba crea las cuentas USUARIO_PRUEBA_1, USUARIO_PRUEBA_2,
// ... del .env (formato nombre|email|contraseña), salvo que
// CARGAR_USUARIOS_DE_PRUEBA sea "false"
func cargarUsuariosDePrueba(queries *sqlc.Queries) {
	if strings.EqualFold(os.Getenv("CARGAR_USUARIOS_DE_PRUEBA"), "false") {
		return
	}

	var usuarios []logic.UsuarioDePrueba
	for i := 1; ; i++ {
		valor := os.Getenv(fmt.Sprintf("USUARIO_PRUEBA_%d", i))
		if valor == "" {
			break
		}
		partes := strings.Split(valor, "|")
		if len(partes) != 3 {
			log.Fatalf("USUARIO_PRUEBA_%d tiene que tener el formato nombre|email|contraseña", i)
		}
		usuarios = append(usuarios, logic.UsuarioDePrueba{
			Nombre:   strings.TrimSpace(partes[0]),
			Email:    strings.TrimSpace(partes[1]),
			Password: strings.TrimSpace(partes[2]),
		})
	}

	creados, err := logic.CargarUsuariosDePrueba(context.Background(), queries, usuarios)
	if err != nil {
		log.Fatalf("Error al cargar los usuarios de prueba: %v", err)
	}
	if creados > 0 {
		log.Printf("Usuarios de prueba cargados: %d (ver .env)", creados)
	}
}
