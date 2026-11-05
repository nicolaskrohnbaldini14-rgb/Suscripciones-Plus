APP_NAME := Tp4-Krohn-Trapani-Cicopiedi
HOST := http://localhost:8080
HURL_FILES := hurl/frontend.hurl hurl/usuario.hurl hurl/servicio.hurl hurl/suscripcion.hurl hurl/pago.hurl hurl/datos_de_prueba.hurl hurl/salud.hurl
# segundos máximos que se espera a que la aplicación esté lista (GET /salud/lista)
ESPERA_APP := 60

.PHONY: all run run-app clean-docker docker-up generate build test test-api docker-down clean


all: test-api

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

generate: docker-up
	@echo "==> Generando código con sqlc..."
	@sqlc generate

build: generate
	@echo "==> Compilando ejecutable..."
	@mkdir -p tmp
	@go build -o ./tmp/$(APP_NAME) .


test:
	@echo "==> Ejecutando pruebas unitarias CRUD..."
	@go test -v -count=1 ./...


test-api: build
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
	echo "==> Probando los endpoints con Hurl..." ; \
	hurl --test --jobs 1 --variable host=$(HOST) $(HURL_FILES) ; \
	STATUS=$$? ; \
	kill $$APP_PID ; \
	$(MAKE) --no-print-directory docker-down ; \
	exit $$STATUS


docker-down:
	@echo "==> Borrando contenedores y volúmenes..."
	@docker compose down -v

run: test-api
	@echo "==> ¡Proceso finalizado con éxito!"

clean:
	@rm -rf tmp

# run-app: levanta la base, genera el código, compila y deja la app corriendo
# en http://localhost:8080. Al cortar con Ctrl+C borra contenedores y volúmenes.
run-app: build
	@echo "==> Aplicación disponible en $(HOST) (Ctrl+C para detener)"
	@trap '$(MAKE) --no-print-directory docker-down' EXIT; \
	trap 'exit 0' INT TERM; \
	./tmp/$(APP_NAME)
