# Suscripciones+ · TP5
Aplicación web en Go para registrar suscripciones y gastos recurrentes
(streaming, música, software, deportes, salud, etc.).


- **Inicio (`/`)**: portada centrada con los botones **Iniciar sesión** y
  **Registrarse**.
- **Barra superior**: el botón **Servicios ▾** despliega, justo debajo, las
  categorías del catálogo 
- **Mi cuenta**: *"Hola, {nombre}"*, cuánto vence en el mes actual y dos
  opciones: **ver las suscripciones** o **crear una nueva**.
- **Mis suscripciones**: las que vencen en el mes (por defecto el actual), con
  buscador, gasto del mes, calendario con los vencimientos, links al mes
  anterior y al siguiente, y el detalle de cada una.

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
git clone https://github.com/nicolaskrohnbaldini14-rgb/Tp1-Krohn-Trapani-Cicopiedi.git
```
```bash
cd Tp1-Krohn-Trapani-Cicopiedi
```
```bash
git checkout tp5
```
```bash
make run-app
```

Luego abrir <http://localhost:8080>. Para detener la aplicación: `Ctrl+C`.

Al iniciar, la aplicación precarga el catálogo de servicios (ver
[Catálogo de servicios]). Para probarla: registrarse,
tocar un servicio en **Servicios ▾** (o ir a *Nueva suscripción*), crear
algunas suscripciones y verlas en *Mis suscripciones*.

`make run-app` realiza todas las tareas necesarias:

1. **Tareas previas**
   - `sqlc generate`: genera el acceso a datos en `db/sqlc/`.
   - `templ generate`: compila los componentes `.templ` a código Go (`*_templ.go`).
   - Compila el ejecutable en `tmp/`.
   - Si no existe `.env`, lo crea copiando `.env.example` (ver
     [Variables de entorno](#variables-de-entorno-env)).
   - Borra contenedores y volúmenes previos (`docker compose down -v`).
   - Levanta PostgreSQL (`docker compose up -d --wait`) y espera a que el
     healthcheck (`pg_isready`) indique que la base está lista. El esquema se
     crea al iniciar el contenedor (`db/schema/schema.sql` se monta en
     `docker-entrypoint-initdb.d`).
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

**Solo los tests de Go** (`main_test.go`, `logic/` y `views/`): `make test`
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

# Arquitectura

```
Navegador ──(GET de páginas, POST de formularios + cookie de sesión)──▶ handlers/ ──▶ logic/ ──▶ db/sqlc ──▶ PostgreSQL
                         ◀──────────── HTML completo ──────────── views/ (templ)
```

- **`views/`** — componentes templ (capa de presentación) y funciones de
  apoyo de las vistas (`vencimientos.go`: meses, calendario, totales).
- **`handlers/`** — handlers HTTP: leen el formulario, verifican la sesión,
  llaman a la lógica y renderizan la vista o redirigen. Reemplazan a los
  handlers JSON del TP4. Como sugiere la clase de lógica del negocio
  ("estructura limpia": handlers separados por rutas, funciones auxiliares y
  manejo centralizado de errores), están separados en archivos:
  - `web.go`: todas las rutas (`Registrar`);
  - `usuarios.go`: inicio, login, registro, logout y mi cuenta;
  - `suscripciones.go`: lista del mes, alta, modificar, ver la contraseña y eliminar;
  - `comunes.go`: lo que comparten las páginas (sesión, layout, avisos,
    renderizado y `describirError`, que traduce los errores de la lógica a
    códigos HTTP);
  - `formularios.go`: lectura de formularios y parámetros, y filtros por mes y
    búsqueda;
  - `sesiones.go`: la cookie de sesión.
- **`logic/`** — reglas de negocio (validaciones, contraseñas con bcrypt,
  catálogo precargado, renovación automática, cuentas distintas, solo
  lectura). Es la misma lógica del TP4, sin la parte HTTP.
- **`db/`** — esquema, queries y código generado por sqlc.



# Documentación de persistencia

El modelo de datos está compuesto por 4 tablas relacionales, definidas en
[`db/schema/schema.sql`](db/schema/schema.sql):

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
  - El servicio tiene que existir; el monto tiene que ser mayor a cero y la fecha de inicio anterior a la de vencimiento.
  - Se crean siempre en estado `Activa`; después se pueden pausar y reactivar desde *Modificar* (`Activa` y `Pausada` son los únicos estados que se pueden elegir). No hay estado `Cancelada`: para dejar una suscripción se la elimina.
  - Usuario y contraseña de la cuenta del servicio son opcionales (un gimnasio o una cancha de pádel no tienen cuenta). La contraseña nunca se muestra salvo que el usuario confirme la suya.
  - Se puede tener **varias suscripciones al mismo servicio** si son cuentas distintas: desde la segunda (mientras la otra siga activa o pausada) hay que indicar el usuario de la cuenta, y no se puede repetir uno en uso (sin distinguir mayúsculas). Esta regla está en `verificarCuentaDistinta`, en `logic/suscripcion.go`.
  - **Renovación automática:** cuando una suscripción activa vence, se renueva sola por el mismo plazo con el que se registró (un mes, un año, etc.). El período vencido queda como historial con estado `Vencida` y se crea uno nuevo con el mismo monto y la misma cuenta. Se hace al iniciar la app y cada vez que se listan las suscripciones. Las pausadas no se renuevan.
  - Las suscripciones `Vencida` y las que vencen en un mes anterior al actual son de **solo lectura**.
  - En el mes actual (y los siguientes), el *Gasto mensual (activo)* suma solo lo que **falta pagar**: las suscripciones activas que todavía no vencieron (`GastoPendiente`). En un mes anterior, el *Gasto mensual* suma todo lo que venció ese mes: las activas y los períodos que ya se renovaron (`TotalActivo`). Las pausadas no suman nunca.
- **Pagos:** la suscripción tiene que existir, el monto tiene que ser mayor a cero y la fecha de pago no puede ser posterior al vencimiento de la suscripción.



# Estructura del proyecto

```
.
├── main.go                 # Conexión, catálogo, renovación y servidor HTTP
├── salud.go                # Health check de la aplicación (/salud/vivo y /salud/lista)
├── main_test.go            # Tests de integración de sqlc
├── handlers/
│   ├── web.go              # Todas las rutas
│   ├── usuarios.go         # Inicio, login, registro, logout y mi cuenta
│   ├── suscripciones.go    # Lista, alta, modificar, contraseña y eliminar
│   ├── comunes.go          # Sesión de la página, renderizado, avisos y errores
│   ├── formularios.go      # Lectura de formularios y filtros
│   └── sesiones.go         # Sesiones con cookie HttpOnly
├── logic/                  # Reglas de negocio sin HTTP: suscripcion.go (incluye cuentas distintas), renovacion.go (renovación y solo lectura), datos_de_prueba.go y sus tests
├── views/                  # Componentes templ, vencimientos.go y views_test.go
├── static/styles.css       # Estilos (sin JavaScript)
├── db/
│   ├── schema/schema.sql
│   ├── queries/queries.sql
│   └── sqlc/               # Generado por sqlc (no se versiona)
├── hurl/                   # Pruebas de la aplicación
├── docker-compose.yml      # PostgreSQL con healthcheck (credenciales del .env)
├── .env.example            # Variables de entorno de ejemplo (el .env no se versiona)
├── .gitignore
├── sqlc.yaml
└── makefile
```
