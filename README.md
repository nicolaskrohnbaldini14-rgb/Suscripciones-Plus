# Suscripciones+ · TP6 (entrega final)
Aplicación web en Go para registrar suscripciones y gastos recurrentes
(streaming, música, software, deportes, salud, etc.).

Las vistas se generan en el servidor con [templ](https://templ.guide) y, desde
el TP6, la app usa [HTMX](https://htmx.org) para **crear y eliminar
suscripciones sin recargar la página**: el servidor devuelve solo los
fragmentos de HTML que cambian y HTMX los pone en su lugar. No escribimos
JavaScript propio y el repositorio no tiene archivos `.js`: HTMX se carga
desde el CDN en el `<head>` del layout.


- **Inicio (`/`)**: portada centrada con los botones **Iniciar sesión** y
  **Registrarse**.
- **Barra superior**: el botón **Servicios ▾** despliega, justo debajo, las
  categorías del catálogo 
- **Mi cuenta**: *"Hola, {nombre}"*, cuánto vence en el mes actual y dos
  opciones: **ver las suscripciones** o **crear una nueva**.
- **Mis suscripciones**: las que vencen en el mes (por defecto el actual), con
  buscador, gasto del mes, calendario con los vencimientos, links al mes
  anterior y al siguiente, y el detalle de cada una. El botón **Eliminar**
  pide confirmación en el aviso negro de la página (no con la ventana del
  navegador) y quita la fila sin recargar la página (HTMX).
- **Nueva suscripción**: el formulario y, debajo, *Tus suscripciones activas*.
  Al crear una, la lista se actualiza y el formulario se vacía sin recargar
  la página (HTMX).

# Requisitos previos

- [Go](https://go.dev/doc/install) 1.25 o superior (si la versión instalada es
  anterior, Go descarga automáticamente la indicada en `go.mod`)
- [Docker](https://docs.docker.com/get-docker/) con Docker Compose
- `make`
- [Hurl](https://hurl.dev/docs/installation.html) (solo para `make test-api`)
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) (opcional: si no
  está instalado, el makefile lo ejecuta con `go run`)


# Ejecución

```bash
git clone https://github.com/nicolaskrohnbaldini14-rgb/Suscripciones-Plus.git
```
```bash
cd Suscripciones-Plus
```
```bash
git checkout tp6
```
```bash
make run-app
```

Luego abrir <http://localhost:8080>. Para detener la aplicación: `Ctrl+C`.

Al iniciar, la aplicación precarga el catálogo de servicios. Para probarla: registrarse
(o entrar con un [usuario de prueba](#usuarios-de-prueba)), tocar un servicio en
**Servicios ▾** (o ir a *Nueva suscripción*), crear algunas suscripciones (la
lista de abajo se actualiza al instante) y verlas en *Mis suscripciones*, donde
se pueden eliminar sin recargar la página.

> Se necesita conexión a internet en el navegador para cargar HTMX desde
> `unpkg.com`.

`make run-app` realiza todas las tareas necesarias:

1. **Tareas previas**
   - `sqlc generate`: genera el acceso a datos en `db/sqlc/`.
   - `templ generate`: compila los componentes `.templ` a código Go (`*_templ.go`).
   - Compila el ejecutable en `tmp/`.
   - Si no existe `.env`, lo crea copiando `.env.example` .
   - Borra contenedores y volúmenes previos (`docker compose down -v`).
   - Levanta PostgreSQL y Prometheus (`docker compose up -d --wait`) y espera
     a que sus healthchecks indiquen que están listos (`pg_isready` para la
     base). El esquema se crea al iniciar el contenedor (`db/schema/schema.sql`
     se monta en `docker-entrypoint-initdb.d`).
2. **Ejecución** del servidor en el puerto 8080.
3. **Tareas posteriores**: al cortar con `Ctrl+C` (o si la app termina) se
   borran los contenedores y volúmenes.
   
## Usuarios de prueba

Para probar la app sin registrarse, el `.env.example` trae tres usuarios que
la app crea al iniciar (si no existen), cada uno con varias suscripciones de
ejemplo:

| Nombre | Email | Contraseña |
|---|---|---|
| Martina | `martina@ejemplo.com` | `martina123` |
| Agustina | `agustina@ejemplo.com` | `agustina123` |
| Nicolás | `nicolas@ejemplo.com` | `nicolas123` |

Las suscripciones se arman a partir del mes actual, así que siempre hay
vencimientos en el mes que se está viendo. Entre las de los tres usuarios hay
suscripciones con y sin cuenta del servicio, dos cuentas distintas de Netflix
(Martina), una del mes anterior que se renueva sola al iniciar (OSDE de Martina),
servicios del hogar (Internet, luz) y seguros, algunas pausadas y otras que
vencen el mes siguiente.

## Otros comandos

| Comando | Qué hace |
|---|---|
| `make` / `make test-api` | Genera, compila, levanta la base, corre los tests de Go y prueba la aplicación con Hurl. Al final borra contenedores y volúmenes. |
| `make generate` | Solo genera el código de sqlc y templ. |
| `make build` | Genera y compila. |
| `make docker-up` | Levanta PostgreSQL esperando el healthcheck. |
| `make test` | Solo los tests de Go (necesita el código generado y la base levantada; ver abajo). |
| `make docker-down` | Borra contenedores y volúmenes manualmente. |
| `make clean` | Borra el ejecutable. |

## Ejecutar pruebas

**Todas las pruebas juntas**: genera el código, compila,
levanta la base, corre los tests de Go, levanta la app, la prueba con Hurl y
al final borra los contenedores y volúmenes.

```bash
make
```

**Solo los tests de Go** (`main_test.go`, `logic/`, `features/` y `plataforma/vista/`): `make test`
necesita el código de sqlc y templ generado y la base levantada, porque
`main_test.go` prueba las consultas contra PostgreSQL. Si se corre solo, falla
con *"Base de datos inalcanzable"* (`make: *** [test] Error 1`). Los pasos son:

```bash
make generate
```
(corre `sqlc generate` y `templ generate`)

```bash
make docker-up
```
(levanta PostgreSQL esperando el healthcheck)

```bash
make test
```

```bash
make docker-down
```
(borra los contenedores y volúmenes)

Los tests vacían las tablas al empezar, así que no conviene correrlos mientras
la app está en uso con datos que se quieran conservar.

# Interfaces dinámicas con HTMX (TP6)

HTMX 1.9.10 se incluye en `plataforma/vista/layout.templ`:

```html
<script src="https://unpkg.com/htmx.org@1.9.10"></script>
```

## Crear una suscripción sin recargar la página

- **Formulario** (`features/nuevasuscripcion/vista.templ`, componente `SuscripcionForm`,
  `id="form-suscripcion"`):
  `hx-post="/cuenta/suscripciones"`, `hx-target="#panel-activas"` y
  `hx-swap="outerHTML"`. `method` y `action` siguen como respaldo.
- **Lista** (`plataforma/vista/activas.templ`, componente `PanelActivas`,
  `id="panel-activas"`): las suscripciones activas. Está debajo del formulario
  en *Nueva suscripción* (y también en *Mi cuenta*).
- **Handler `POST /cuenta/suscripciones`** (`crearSuscripcion` en
  `features/nuevasuscripcion/handler.go`): **ya no redirige**. Crea la suscripción, vuelve
  a consultar la lista y devuelve solo el fragmento `RespuestaSuscripcionCreada`:
  1. `PanelActivas` actualizado: reemplaza a `#panel-activas` (el `hx-target`).
  2. `SuscripcionFormOOB`: el formulario **vacío** con `hx-swap-oob="true"`.
     HTMX reemplaza el formulario con el mismo id, así se limpian los campos
     (*Out-of-Band Swap*).
  3. El aviso *"Suscripción creada."*, también fuera de banda (`#avisos`).
- **Errores** (servicio sin elegir, monto inválido, cuenta repetida…): HTMX
  1.9 no reemplaza nada si la respuesta es 4xx/5xx. Por eso, cuando el pedido
  viene de HTMX (encabezado `HX-Request`), el servidor responde 200 con el
  formulario con lo que se escribió y el aviso de error, y cambia el destino
  con `HX-Retarget: #form-suscripcion` y `HX-Reswap: outerHTML`. Sin HTMX
  (por ejemplo, en las pruebas con Hurl) se mantiene el código de error (400,
  409…) y la página completa.
- **Catálogo**: el botón *Servicio* del formulario abre el catálogo grande
  (`formaction="/cuenta/suscripciones/nueva/catalogo"`). Como el formulario se
  envía con HTMX, ese handler responde con `HX-Retarget: body` y
  `HX-Reswap: innerHTML`: se redibuja la página con el catálogo abierto sin
  perder lo escrito.

## Eliminar una suscripción sin recargar la página

- **Consulta sqlc**: `DeleteSubscripcion` en `db/queries/queries.sql`
  (`DELETE FROM suscripcion WHERE id_suscripcion = $1`).
- **Confirmación sin la ventana del navegador.** La consigna sugiere
  `hx-confirm` (opcional), pero ese atributo abre la ventana nativa del
  navegador (*"localhost:8080 dice…"*). En su lugar, la pregunta aparece en el
  mismo aviso negro de los mensajes, también con HTMX y sin JavaScript propio:
  - **Botón *Eliminar*** en cada fila de *Mis suscripciones*
    (`ItemSuscripcion` en `features/missuscripciones/lista.templ`; cada fila es un
    `<li id="fila-{id}">`): `hx-get="/cuenta/suscripciones/{id}/confirmar-eliminar?mes=AAAA-MM"`,
    `hx-target="#avisos"`, `hx-swap="outerHTML"`.
  - **Handler `GET /cuenta/suscripciones/{id}/confirmar-eliminar`**
    (`confirmarEliminar`): verifica que la suscripción sea del usuario y que
    no sea de solo lectura, y devuelve el aviso *"¿Seguro que querés eliminar
    la suscripción a …? No se puede deshacer."* (`ConfirmarEliminar`), fijo
    arriba de la pantalla, con dos botones:
    - ***Sí, eliminar***: `hx-delete="/cuenta/suscripciones/{id}?mes=AAAA-MM"`,
      `hx-target="#fila-{id}"` (el `<li>` de esa suscripción) y
      `hx-swap="outerHTML"`.
    - ***Cancelar***: pide el mismo handler con `cancelar=1`, que devuelve el
      aviso vacío.
- **Handler `DELETE /cuenta/suscripciones/{id}`** (`eliminar`): lee el id de la
  ruta, verifica que la suscripción sea del usuario (si no, 404) y que no sea
  de solo lectura (si no, 403), la borra y responde **200 sin contenido
  principal**: HTMX reemplaza la fila por nada y desaparece. En la misma
  respuesta, fuera de banda (`hx-swap-oob`), llegan el *Gasto mensual*
  (`#total-mes`) y el calendario (`#panel-calendario`) recalculados, y el
  aviso *"Suscripción eliminada."*. El `mes` (y la búsqueda `q`) viajan en
  la URL para poder recalcularlos. Si era la última suscripción de la lista,
  se reemplaza la lista entera para mostrar el mensaje de lista vacía.

## Otros detalles

- Si la sesión venció y el pedido es de HTMX, `conSesion` responde con
  `HX-Redirect: /login`, así HTMX lleva al navegador entero al login (en vez
  de meter la página de login dentro del fragmento).
- Respuestas por ruta:

| Ruta | Sin HTMX | Con HTMX (`HX-Request: true`) |
|---|---|---|
| `POST /cuenta/suscripciones` (ok) | 200 · fragmento | 200 · `PanelActivas` + formulario vacío (oob) + aviso (oob) |
| `POST /cuenta/suscripciones` (error) | 400/409 · página completa | 200 · formulario + aviso (oob), `HX-Retarget: #form-suscripcion` |
| `POST /cuenta/suscripciones/nueva/catalogo` | 200 · página completa | 200 · página completa, `HX-Retarget: body` |
| `DELETE /cuenta/suscripciones/{id}` (ok) | 200 · vacío + gasto, calendario y aviso (oob) | igual |
| `DELETE /cuenta/suscripciones/{id}` (ajena / solo lectura) | 404 / 403 · aviso | 200 · aviso, `HX-Retarget: #avisos` |

# Monitoreo con Prometheus

Siguiendo la clase de monitoreo, la app expone sus métricas y `make run-app`
levanta también un servidor de **Prometheus** (servicio `prometheus` del
`docker-compose.yml`) que las va a buscar cada 5 segundos.

- **`GET /metrics`** devuelve las métricas en el formato de texto de
  Prometheus (cliente oficial `github.com/prometheus/client_golang`). El
  código está en [`plataforma/metricas.go`](plataforma/metricas.go).
- Un **middleware** (`plataforma.Metricas`) envuelve todas las rutas y mide
  cada pedido una sola vez, con el marco **RED**:

  | Métrica | Tipo | Labels | Para qué |
  |---|---|---|---|
  | `http_requests_total` | Counter | `method`, `path`, `status` | Cuántos pedidos llegan (*Rate*) y cuántos fallan (*Errors*, por el código) |
  | `http_request_duration_seconds` | Histogram | `method`, `path` | Cuánto tardan (*Duration*) |

  Como `path` se usa el **patrón de la ruta** (por ejemplo
  `DELETE /cuenta/suscripciones/{id}`) y no la URL con el id, para no crear
  una serie por cada URL (baja cardinalidad). Las URLs que no existen quedan
  como `sin ruta`.
- **Métricas del negocio**: `suscripciones_creadas_total`,
  `suscripciones_eliminadas_total` e `inicios_de_sesion_total{resultado="ok|error"}`.
- El cliente agrega solas las métricas de Go y del proceso (`go_goroutines`,
  memoria, CPU).
- **Alertas** ([`monitoreo/alertas.yml`](monitoreo/alertas.yml)): `AppCaida`
  (la app no responde en `/metrics` durante 30 s) y `MuchosErroresInternos`
  (más del 5 % de los pedidos de los últimos 5 minutos terminan en 5xx
  durante 2 minutos). Se ven en <http://localhost:9090/alerts>; no hay
  Alertmanager, así que no se envían avisos.

Para verlo: `make run-app`, usar la app y abrir <http://localhost:9090>.
Consultas PromQL de ejemplo (las de la clase):

```
rate(http_requests_total[5m])
sum by (path, status) (http_requests_total{job="suscripciones"})
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))
suscripciones_creadas_total
```

La configuración de Prometheus está en
[`monitoreo/prometheus.yml`](monitoreo/prometheus.yml). Como la app corre en
la máquina (fuera de Docker), Prometheus la busca en `host.docker.internal:8080`.

# Arquitectura

## Vertical Slice Architecture

Siguiendo la clase de diseño (*Arquitectura de Rebanadas Verticales*), el
código web está organizado **por funcionalidad** y no por capa técnica. Cada
*slice* en `features/` tiene todo lo que necesita su funcionalidad de punta a
punta: sus rutas (`Registrar`), su handler y su vista templ con los datos
(DTOs) que muestra. Para cambiar, por ejemplo, cómo se elimina una suscripción,
se tocan solo los archivos de `features/missuscripciones/`.

| Slice | Rutas | Qué hace |
|---|---|---|
| `features/inicio` | `GET /` | Portada pública |
| `features/sesion` | `GET/POST /login`, `POST /logout` | Iniciar y cerrar sesión |
| `features/registro` | `GET/POST /registro` | Crear una cuenta |
| `features/cuenta` | `GET /cuenta` | Saludo, lo que vence este mes y las suscripciones activas |
| `features/nuevasuscripcion` | `GET /cuenta/suscripciones/nueva`, `POST /cuenta/suscripciones/nueva/catalogo`, `POST /cuenta/suscripciones` | Formulario, catálogo para elegir el servicio y alta con HTMX |
| `features/missuscripciones` | `GET /cuenta/suscripciones`, `POST .../{id}/modificar`, `POST .../{id}/password`, `GET .../{id}/confirmar-eliminar`, `DELETE .../{id}` | Lista del mes, buscador, calendario, detalle, modificar, ver la contraseña y eliminar con HTMX |

`main.go` solo llama al `Registrar` de cada slice. Los slices **no se
importan entre sí**; lo que comparten está aparte, como propone la filmina
*"Slices: ¿Y el código compartido?"*:

- **`plataforma/`** — el código transversal (lo que la filmina llama
  `internal/platform`): `App` (los servicios de `logic/` con las consultas de
  sqlc y las sesiones, `ConSesion` para exigir el login y `Pagina` para los
  datos del layout), `sesiones.go` (cookie HttpOnly), `http.go` (renderizado,
  avisos, ayudas de HTMX y `DescribirError`, que traduce los errores de la
  lógica a códigos HTTP), `formularios.go` (lectura de formularios y filtro
  por mes) y `metricas.go` (monitoreo con Prometheus).
- **`plataforma/vista/`** — la interfaz que comparten todas las páginas:
  `Layout`, la barra con el catálogo de servicios, los avisos, los íconos, el
  panel de suscripciones activas (lo usan *Mi cuenta* y *Nueva suscripción*) y
  el formato de montos, fechas y meses.
- **`logic/`** — las reglas de negocio, que funcionan como el *Shared Kernel*
  de la filmina: la renovación automática, el solo lectura, las cuentas
  distintas del mismo servicio y la validación del monto las usan varios
  slices (crear, modificar, listar) y también `main.go` al iniciar, así que
  no pertenecen a uno solo.
- **`db/`** — esquema, queries y código generado por sqlc.

Los tests de cada vista están dentro de su slice (`features/*/vista_test.go`)
y usan las ayudas de `plataforma/pruebas/`. Las rutas no cambiaron, así que
las pruebas con Hurl son las mismas.



# Documentación de persistencia

El modelo de datos está compuesto por 4 tablas relacionales, definidas en
[`db/schema/schema.sql`]

| Tabla | Descripción |
|---|---|
| `usuario` | Cuenta de la persona que usa la app (nombre, email y contraseña hasheada con bcrypt). |
| `servicio` | Catálogo de servicios disponibles (nombre + categoría), normalizado para poder agrupar el gasto por categoría. Se precarga al iniciar la app. |
| `suscripcion` | Relaciona un `usuario` con un `servicio`: monto, fecha de inicio, fecha de vencimiento, credenciales de acceso al servicio (usuario/contraseña de esa cuenta, **opcionales**: pueden ser `NULL`) y estado (`Activa`, `Pausada` o `Vencida`). |
| `pago` | Historial real de cobros de una suscripción. Permite calcular el gasto efectivo mes a mes, no solo el proyectado. |


# Reglas de negocio

- **Usuarios:** nombre, email y password son obligatorios. El email tiene que tener un formato válido y no puede repetirse. La password se guarda hasheada y nunca se muestra.
- **Acceso:** cada usuario ve y modifica únicamente sus propias suscripciones.
- **Servicios:** el catálogo viene precargado y es de solo lectura.
- **Suscripciones:**
  - El servicio tiene que existir; el monto tiene que ser mayor a cero y de hasta $ 99.999.999,99 (lo máximo que entra en la columna `DECIMAL(10,2)`; si no, se avisa "ingrese un monto válido") y la fecha de inicio anterior a la de vencimiento.
  - Se crean siempre en estado `Activa`; después se pueden pausar y reactivar desde *Modificar* (`Activa` y `Pausada` son los únicos estados que se pueden elegir). No hay estado `Cancelada`: para dejar una suscripción se la elimina.
  - Usuario y contraseña de la cuenta del servicio son opcionales (un gimnasio o una cancha de pádel no tienen cuenta). La contraseña nunca se muestra salvo que el usuario confirme la suya.
  - Se puede tener **varias suscripciones al mismo servicio** si son cuentas distintas: desde la segunda (mientras la otra siga activa o pausada) hay que indicar el usuario de la cuenta, y no se puede repetir uno en uso (sin distinguir mayúsculas). Esta regla está en `verificarCuentaDistinta`, en `logic/suscripcion.go`.
  - **Renovación automática:** cuando una suscripción activa vence, se renueva sola por el mismo plazo con el que se registró (un mes, un año, etc.). El período vencido queda como historial con estado `Vencida` y se crea uno nuevo con el mismo monto y la misma cuenta. Se hace al iniciar la app y cada vez que se listan las suscripciones. Las pausadas no se renuevan.
  - Las suscripciones `Vencida` y las que vencen en un mes anterior al actual son de **solo lectura**.
  - En el mes actual (y los siguientes), el *Gasto mensual (activo)* suma solo lo que **falta pagar**: las suscripciones activas que todavía no vencieron (`GastoPendiente`). En un mes anterior, el *Gasto mensual* suma todo lo que venció ese mes: las activas y los períodos que ya se renovaron (`TotalActivo`). Las pausadas no suman nunca.
- **Pagos:** la suscripción tiene que existir, el monto tiene que ser mayor a cero y de hasta $ 99.999.999,99 y la fecha de pago no puede ser posterior al vencimiento de la suscripción.



# Estructura del proyecto

```
.
├── main.go                 # Conexión, catálogo, renovación y servidor HTTP
├── salud.go                # Health check de la aplicación (/salud/vivo y /salud/lista)
├── monitoreo/              # Configuración de Prometheus (prometheus.yml) y reglas de alerta (alertas.yml)
├── main_test.go            # Tests de integración de sqlc
├── features/               # Vertical slices: cada uno con handler.go (rutas y handlers), vista .templ y tests
│   ├── inicio/             # Portada
│   ├── sesion/             # Login y logout
│   ├── registro/           # Crear cuenta
│   ├── cuenta/             # Mi cuenta
│   ├── nuevasuscripcion/   # Formulario, catálogo y alta (HTMX)
│   └── missuscripciones/   # Lista, calendario, detalle, modificar, contraseña y eliminar (HTMX)
├── plataforma/             # Código compartido por los slices: App, sesiones, http.go, formularios.go y metricas.go (Prometheus)
│   ├── vista/              # Layout, barra, avisos, panel de activas y formato (templ)
│   └── pruebas/            # Ayudas para los tests de las vistas
├── logic/                  # Reglas de negocio sin HTTP (Shared Kernel): suscripcion.go (incluye cuentas distintas), renovacion.go (renovación y solo lectura), datos_de_prueba.go y sus tests
├── static/styles.css       # Estilos (sin JavaScript propio; HTMX se carga desde el CDN)
├── db/
│   ├── schema/schema.sql
│   ├── queries/queries.sql
│   └── sqlc/               # Generado por sqlc (no se versiona)
├── hurl/                   # Pruebas de la aplicación
├── docker-compose.yml      # PostgreSQL con healthcheck (credenciales del .env) y Prometheus
├── .env.example            # Variables de entorno de ejemplo (el .env no se versiona)
├── .gitignore
├── sqlc.yaml
└── makefile
```
