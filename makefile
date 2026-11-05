APP_NAME := Tp5-Krohn-Trapani-Cicopiedi
HOST := http://localhost:8080
HURL_FILES := hurl/publico.hurl hurl/cuentas.hurl hurl/suscripcion.hurl hurl/datos_de_prueba.hurl hurl/salud.hurl
# segundos máximos que se espera a que la aplicación esté lista (GET /salud/lista)
ESPERA_APP := 60

# Si sqlc no está instalado se compila y ejecuta con "go run"
SQLC := $(shell command -v sqlc 2>/dev/null || echo "go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1")
# templ está declarado como herramienta en go.mod (directiva "tool")
TEMPL := go tool templ

.PHONY: all run-app run test test-api generate build clean-docker docker-up docker-down clean


all: test-api

# ---------------------------------------------------------------------------
# Generación de código y compilación
# ---------------------------------------------------------------------------

generate:
	@echo "==> Generando código con sqlc..."
	@$(SQLC) generate
	@echo "==> Generando vistas con templ..."
	@$(TEMPL) generate

build: generate
	@echo "==> Compilando ejecutable..."
	@mkdir -p tmp
	@go build -o ./tmp/$(APP_NAME) .

# ---------------------------------------------------------------------------
# Base de datos (Docker)
# ---------------------------------------------------------------------------

# si no existe el .env (credenciales), se crea a partir de .env.example
.env:
	@echo "==> Creando .env a partir de .env.example..."
	@cp .env.example .env

clean-docker: .env
	@echo "==> Limpiando contenedores y volúmenes previos..."
	@-docker compose down -v 2>/dev/null || true

docker-up: clean-docker
	@echo "==> Levantando PostgreSQL en Docker..."
	@docker compose up -d --wait
	@echo "==> La base de datos está lista (healthcheck OK)"

docker-down:
	@echo "==> Borrando contenedores y volúmenes..."
	@docker compose down -v

# ---------------------------------------------------------------------------
# Ejecución
# ---------------------------------------------------------------------------

# run-app: tareas previas (sqlc, templ, compilación, limpieza y arranque de la
# base esperando el healthcheck), ejecución de la app y, al cortar con Ctrl+C,
# tareas posteriores (borrado de contenedores y volúmenes).
run-app: build docker-up
	@echo "==> Aplicación disponible en $(HOST) (Ctrl+C para detener)"
	@trap '$(MAKE) --no-print-directory docker-down' EXIT; \
	trap 'exit 0' INT TERM; \
	./tmp/$(APP_NAME)

run: run-app

# ---------------------------------------------------------------------------
# Pruebas
# ---------------------------------------------------------------------------

test:
	@echo "==> Ejecutando pruebas de Go (sqlc + vistas)..."
	@go test -v -count=1 ./...

# test-api: corre los tests de Go y luego prueba la aplicación con Hurl
test-api: build docker-up
	@$(MAKE) --no-print-directory test || { $(MAKE) --no-print-directory docker-down; exit 1; }
	@echo "==> Levantando la aplicación..."
	@./tmp/$(APP_NAME) & APP_PID=$$! ; \
	echo "==> Esperando el health check de la aplicación (hasta $(ESPERA_APP) s)..." ; \
	ESPERA=0 ; \
	until curl -sf -o /dev/null $(HOST)/salud/lista ; do \
		if ! kill -0 $$APP_PID 2>/dev/null ; then \
			echo "La aplicación terminó antes de responder" ; $(MAKE) --no-print-directory docker-down ; exit 1 ; \
		fi ; \
		if [ $$ESPERA -ge $(ESPERA_APP) ] ; then \
			echo "La aplicación no quedó lista en $(ESPERA_APP) s" ; kill $$APP_PID ; $(MAKE) --no-print-directory docker-down ; exit 1 ; \
		fi ; \
		ESPERA=$$((ESPERA + 1)) ; sleep 1 ; \
	done ; \
	echo "==> Probando la aplicación con Hurl..." ; \
	hurl --test --jobs 1 --variable host=$(HOST) $(HURL_FILES) ; \
	STATUS=$$? ; \
	kill $$APP_PID ; \
	$(MAKE) --no-print-directory docker-down ; \
	exit $$STATUS

clean:
	@rm -rf tmp
