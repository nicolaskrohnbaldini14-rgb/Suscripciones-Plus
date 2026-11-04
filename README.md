# Suscripciones+ · TP5 — Renderizado del lado del servidor con templ

Aplicación web en Go para registrar suscripciones y gastos recurrentes
(streaming, música, software, deportes, salud, etc.). En este TP se reemplazó
el frontend del TP4 (HTML + JavaScript que consumía una API JSON) por **vistas
renderizadas en el servidor con [templ](https://templ.guide/)**. El acceso a
datos sigue siendo con [sqlc](https://sqlc.dev/) sobre PostgreSQL en Docker.

La entrega **no contiene archivos JavaScript**: todo se resuelve con HTML,
CSS y formularios clásicos con recargas de página completas. La aplicación
tiene las mismas funciones que la versión final del TP4:

- **Inicio (`/`)**: portada centrada con los botones **Iniciar sesión** y
  **Registrarse**.
- **Barra superior**: el botón **Servicios ▾** abre un panel a pantalla
  completa con el catálogo agrupado por categoría (cada una con su logo). Con
  la sesión iniciada, cada servicio lleva a suscribirse a él.
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

templ **no** hace falta instalarlo: está declarado como herramienta en `go.mod`
(directiva `tool`) y se ejecuta con `go tool templ`.

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
[Catálogo de servicios](#catálogo-de-servicios)). Para probarla: registrarse,
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

## Variables de entorno (`.env`)

Las credenciales de la base no están escritas en el código: se leen de un
archivo `.env` que **no se sube al repositorio** (está en `.gitignore`). En el
repo está [`.env.example`](.env.example) con valores de ejemplo para
desarrollo; `make run-app` (y `make`) lo copian como `.env` si todavía no
existe. Para usar otros valores:

```bash
cp .env.example .env
```

y editar `.env`. Lo usan:

- `docker-compose.yml` (`env_file: .env`): usuario, contraseña y nombre de la
  base del contenedor, y el puerto publicado (`POSTGRES_PORT`).
- La aplicación y los tests de Go (`cadenaConexion()` en `main.go`): cargan el
  `.env` con [godotenv](https://github.com/joho/godotenv) y arman la URL de
  conexión. Si una variable ya está definida en el sistema, tiene prioridad.

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

Se definen en el `.env` como `USUARIO_PRUEBA_1`, `USUARIO_PRUEBA_2`, ... con
el formato `nombre|email|contraseña`; se pueden agregar más o desactivarlos
con `CARGAR_USUARIOS_DE_PRUEBA=false`. La carga está en
[`logic/datos_de_prueba.go`](logic/datos_de_prueba.go) y usa los mismos
servicios de la app, así que pasa por las mismas validaciones y las
contraseñas se guardan hasheadas.

## Esperas

- **Base de datos:** `docker compose up --wait` espera el healthcheck
  (`pg_isready`), que se reintenta durante unos 2 minutos. Además, la
  aplicación reintenta conectarse cada un segundo durante 60 segundos antes de
  rendirse, por si la base tarda en aceptar conexiones.
- **Aplicación (pruebas):** antes de correr Hurl, el makefile consulta la app
  cada un segundo hasta que responda, con un máximo de `ESPERA_APP` segundos
  (60 por defecto). Si la app se cae o no responde a tiempo, lo informa y borra
  los contenedores.

## Otros comandos

| Comando | Qué hace |
|---|---|
| `make` / `make test-api` | Genera, compila, levanta la base, corre los tests de Go y prueba la aplicación con Hurl. Al final borra contenedores y volúmenes. |
| `make generate` | Solo genera el código de sqlc y templ. |
| `make build` | Genera y compila. |
| `make docker-down` | Borra contenedores y volúmenes manualmente. |
| `make clean` | Borra el ejecutable. |

# Arquitectura

```
Navegador ──(GET de páginas, POST de formularios + cookie de sesión)──▶ handlers/ ──▶ logic/ ──▶ db/sqlc ──▶ PostgreSQL
                         ◀──────────── HTML completo ──────────── views/ (templ)
```

- **`views/`** — componentes templ (capa de presentación) y funciones de
  apoyo de las vistas (`vencimientos.go`: meses, calendario, totales).
- **`handlers/`** — handlers HTTP (`web.go`) y sesiones (`sesiones.go`): leen
  el formulario, verifican la sesión, llaman a la lógica y renderizan la
  vista o redirigen. Reemplazan a los handlers JSON del TP4.
- **`logic/`** — reglas de negocio (validaciones, contraseñas con bcrypt,
  catálogo precargado, renovación automática, cuentas distintas, solo
  lectura). Es la misma lógica del TP4, sin la parte HTTP.
- **`db/`** — esquema, queries y código generado por sqlc.

## Sesiones

- Las contraseñas se guardan **hasheadas con bcrypt** (`golang.org/x/crypto/bcrypt`).
- Al registrarse o iniciar sesión se genera un token aleatorio que se guarda en
  memoria del servidor y se envía en una cookie `HttpOnly` y `SameSite=Lax`
  llamada `sesion` (dura 24 h). Si se reinicia el servidor hay que volver a ingresar.
- Las páginas de la cuenta exigen sesión (`conSesion`): si no la hay,
  redirigen a `/login`. El usuario de cada suscripción sale de la sesión, y
  cada uno ve y modifica solo las suyas (las ajenas responden `404`).

## Cómo se resuelve cada función sin JavaScript

| Función | Cómo |
|---|---|
| Panel de servicios | CSS: se abre con `:hover` / `:focus-within` sobre **Servicios ▾** y queda abierto mientras el cursor esté encima. Muestra solo las categorías; los servicios de cada una se despliegan al pasar el cursor por ella (o al tocarla), también con `:hover` / `:focus-within`. |
| Elegir el servicio desde el formulario | *Elegir desde el catálogo* abre `?catalogo=1`, que dibuja el panel abierto en modo "elegir" con el servicio actual resaltado. |
| Vencimientos por mes | `?mes=AAAA-MM`; el servidor filtra y arma los links al mes anterior y al siguiente. |
| Buscador | Formulario `GET ?q=`: el servidor filtra por servicio, categoría o cuenta. |
| Calendario | Se arma en Go (`ArmarCalendario`); el cartel de cada día marcado se muestra con CSS al pasar el cursor. |
| Detalle de una suscripción | `?ver={id}`: el servidor dibuja el recuadro sobre la lista desenfocada; la × y el fondo son links que lo cierran. |
| Modificar (monto, usuario, contraseña y estado) | `?ver={id}&editar=1` muestra el formulario, con un selector *Activa* / *Pausada*; `POST .../{id}/modificar`. En la lista el estado solo se muestra. |
| Ver la contraseña de la cuenta | `<details>` con un formulario que pide la contraseña del usuario (`POST .../{id}/password`); el servidor la muestra solo si es correcta. |
| Eliminar | Formulario `POST .../{id}/eliminar`. |
| Avisos | Después de redirigir, `?ok=creada` (etc.) muestra el aviso; los errores se muestran directamente con su código HTTP. |

## Componentes templ

| Archivo | Componentes | Descripción |
|---|---|---|
| `layout.templ` | `Layout`, `Navegacion`, `PanelServicios` | Estructura base (`<!DOCTYPE>`, `<head>`, `<body>`) con el link a la hoja de estilos, la barra superior y el panel del catálogo. |
| `home.templ` | `HomePage` | Portada centrada. |
| `login.templ` | `LoginPage` | Inicio de sesión. |
| `usuario_form.templ` | `RegistroPage`, `UsuarioForm` | Registro de usuarios. |
| `cuenta.templ` | `CuentaPage` | *"Hola, {nombre}"*, resumen del mes y las dos opciones. |
| `suscripcion_list.templ` | `MisSuscripcionesPage`, `ItemSuscripcion`, `Calendario`, `DetalleSuscripcion` | Lista del mes (logo de la categoría, monto, *Modificar*, estado y *Eliminar*), buscador, calendario y detalle. |
| `suscripcion_form.templ` | `NuevaSuscripcionPage`, `SuscripcionForm` | Alta de suscripción (servicio agrupado por categoría, monto, fechas y credenciales opcionales). |
| `componentes.templ` | `AvisoMensaje`, `VacioLista`, `IconoCategoria`, `ErrorPage` | Componentes reutilizables, íconos SVG de las categorías y formato de montos y fechas. |

Los estilos están en [`static/styles.css`](static/styles.css): la paleta de
la primera entrega (azul acero `#567c8d`, azul petróleo `#002f44`/`#004b6e` y
crema `#ece6d4`) con una textura de grilla suave en el fondo, layout con
**CSS Grid** y **Flexbox**, y avisos con fondo negro.

## Rutas

| Método | Ruta | Acceso | Descripción |
|---|---|---|---|
| `GET` | `/` | Público | Portada. |
| `GET` / `POST` | `/login` | Público | Formulario / inicio de sesión. |
| `GET` / `POST` | `/registro` | Público | Registro (deja la sesión iniciada). |
| `POST` | `/logout` | Con sesión | Cierra la sesión. |
| `GET` | `/cuenta` | Con sesión | *"Hola, {nombre}"* y resumen del mes. |
| `GET` | `/cuenta/suscripciones` | Con sesión | Suscripciones del mes. Parámetros: `mes`, `q` (búsqueda), `ver` (detalle), `editar`, `ok` (aviso). |
| `GET` | `/cuenta/suscripciones/nueva` | Con sesión | Formulario de alta. Parámetros: `servicio` (preseleccionado), `catalogo=1`. |
| `POST` | `/cuenta/suscripciones` | Con sesión | Crea la suscripción y redirige al mes en que vence. |
| `POST` | `/cuenta/suscripciones/{id}/modificar` | Con sesión | Modifica monto, usuario y contraseña de la cuenta y el estado (*Activa* o *Pausada*). |
| `POST` | `/cuenta/suscripciones/{id}/password` | Con sesión | Muestra la contraseña de la cuenta si se confirma la del usuario. |
| `POST` | `/cuenta/suscripciones/{id}/eliminar` | Con sesión | Elimina la suscripción. |
| `GET` | `/static/styles.css` | Público | Hoja de estilos (`Cache-Control: no-cache`). |

### Patrón Post/Redirect/Get

Los handlers `POST` parsean el formulario (`r.ParseForm()`), convierten los
valores (ids, montos y fechas de `<input type="date">`) y llaman a la capa de
lógica, que usa los métodos `Create...`/`Update...`/`Delete...` de sqlc:

- **Si sale bien** responden `303 See Other` (`http.Redirect`) hacia la
  página que muestra el resultado, con un `?ok=` que indica el aviso. El
  navegador la pide de nuevo y se ve la lista actualizada; recargar no
  reenvía el formulario.
- **Si falla** se vuelve a renderizar la página con el mensaje de error y el
  código HTTP correspondiente (conservando lo que se había escrito): `400`
  datos inválidos, `401` credenciales incorrectas, `403` suscripción de solo
  lectura o contraseña de confirmación incorrecta, `404` registro inexistente
  o ajeno, `409` email repetido o cuenta repetida.
- Ver la contraseña de la cuenta no redirige: la página se renderiza en la
  respuesta del `POST`, así la contraseña nunca queda en la URL.

# Documentación de persistencia

El modelo de datos está compuesto por 4 tablas relacionales, definidas en
[`db/schema/schema.sql`](db/schema/schema.sql):

| Tabla | Descripción |
|---|---|
| `usuario` | Cuenta de la persona que usa la app (nombre, email y contraseña hasheada con bcrypt). |
| `servicio` | Catálogo de servicios disponibles (nombre + categoría), normalizado para poder agrupar el gasto por categoría. Se precarga al iniciar la app. |
| `suscripcion` | Relaciona un `usuario` con un `servicio`: monto, fecha de inicio, fecha de vencimiento, credenciales de acceso al servicio (usuario/contraseña de esa cuenta, **opcionales**: pueden ser `NULL`) y estado (`Activa`, `Pausada` o `Vencida`). |
| `pago` | Historial real de cobros de una suscripción. Permite calcular el gasto efectivo mes a mes, no solo el proyectado. |

**Relaciones:**

<img width="1024" height="559" alt="ModeloRelacionalWEB" src="https://github.com/user-attachments/assets/a5284446-98ce-4dc8-b4a0-a3bec07d5b3e" />

- `suscripcion.id_usuario` → `usuario.id_usuario` (`ON DELETE CASCADE`)
- `suscripcion.id_servicio` → `servicio.id_servicio`
- `pago.id_suscripcion` → `suscripcion.id_suscripcion` (`ON DELETE CASCADE`)

El acceso a datos se genera con sqlc a partir de
[`db/queries/queries.sql`](db/queries/queries.sql). Las credenciales de la
cuenta son `*string` en Go (override en `sqlc.yaml`) para poder ser `NULL`.
Además de las queries CRUD hay:

- `CrearServicioSiNoExiste`: precarga el catálogo sin duplicar.
- `ListSuscripcionesVigentes`: suscripciones activas o pausadas de un usuario
  a un servicio (para exigir cuentas distintas).
- `ListSuscripcionesParaRenovar` y `RenovarSuscripcion`: renovación automática
  en una sola sentencia SQL atómica.

La tabla `pago` y su lógica se mantienen del TP3, pero la interfaz no los
muestra. El código generado (`db/sqlc/`) y los `*_templ.go` **no se
versionan**: se regeneran con `make generate` (incluido en `make run-app`).

## Catálogo de servicios

Al iniciar, `main.go` llama a `CargarCatalogoInicial` (`logic/servicio.go`),
que inserta los servicios que todavía no existan. No se pueden crear ni
borrar servicios desde la aplicación.

| Categoría | Servicios |
|---|---|
| Streaming | Netflix, HBO Max, Disney+, Prime Video, Paramount+, Star+, YouTube Premium |
| Música | Spotify, iTunes, YouTube Music, Deezer, Tidal |
| Software | Microsoft 365, Google One, iCloud+, Adobe Creative Cloud, ChatGPT Plus, Canva Pro, Dropbox |
| Hogar | Internet / Wifi, Luz, Gas, Agua, Cable / TV, Telefonía fija, Expensas |
| Seguros | Seguro de auto, Seguro de moto, Seguro de hogar, Seguro de vida, Seguro de celular, Seguro de viaje |
| Deportes | Gimnasio, Pádel, Fútbol, Tenis, Natación, Yoga, Crossfit, Club deportivo |
| Salud | OSDE, Swiss Medical, Galeno, Medifé, Sancor Salud, Omint |
| Financiero | Mantenimiento de cuenta bancaria, Cuota de préstamo, Tarjeta de crédito |
| Turismo | Aerolíneas Plus, Despegar, Booking.com, Airbnb |
| Otros | Telefonía celular, Mercado Libre Meli+, Rappi Pro, PedidosYa Plus |

# Reglas de negocio

- **Usuarios:** nombre, email y password son obligatorios. El email tiene que tener un formato válido y no puede repetirse. La password se guarda hasheada y nunca se muestra.
- **Acceso:** cada usuario ve y modifica únicamente sus propias suscripciones.
- **Servicios:** el catálogo viene precargado y es de solo lectura.
- **Suscripciones:**
  - El servicio tiene que existir; el monto tiene que ser mayor a cero y la fecha de inicio anterior a la de vencimiento.
  - Se crean siempre en estado `Activa`; después se pueden pausar y reactivar desde *Modificar* (`Activa` y `Pausada` son los únicos estados que se pueden elegir). No hay estado `Cancelada`: para dejar una suscripción se la elimina.
  - Usuario y contraseña de la cuenta del servicio son opcionales (un gimnasio o una cancha de pádel no tienen cuenta). La contraseña nunca se muestra salvo que el usuario confirme la suya.
  - Se puede tener **varias suscripciones al mismo servicio** si son cuentas distintas: desde la segunda (mientras la otra siga activa o pausada) hay que indicar el usuario de la cuenta, y no se puede repetir uno en uso (sin distinguir mayúsculas).
  - **Renovación automática:** cuando una suscripción activa vence, se renueva sola por el mismo plazo con el que se registró (un mes, un año, etc.). El período vencido queda como historial con estado `Vencida` y se crea uno nuevo con el mismo monto y la misma cuenta. Se hace al iniciar la app y cada vez que se listan las suscripciones. Las pausadas no se renuevan.
  - Las suscripciones `Vencida` y las que vencen en un mes anterior al actual son de **solo lectura**.
  - En el mes actual (y los siguientes), el *Gasto mensual (activo)* suma solo lo que **falta pagar**: las suscripciones activas que todavía no vencieron (`GastoPendiente`). En un mes anterior, el *Gasto mensual* suma todo lo que venció ese mes: las activas y los períodos que ya se renovaron (`TotalActivo`). Las pausadas no suman nunca.
- **Pagos:** la suscripción tiene que existir, el monto tiene que ser mayor a cero y la fecha de pago no puede ser posterior al vencimiento de la suscripción.

# Pruebas

`make` (o `make test-api`) corre:

1. **Tests de Go** (`go test ./...`):
   - `main_test.go`: CRUD de cada tabla con sqlc contra PostgreSQL en Docker
     (incluida una suscripción sin credenciales).
   - `logic/renovacion_test.go`: cálculo del plazo de renovación, regla de
     solo lectura y comparación de cuentas.
   - `views/views_test.go`: renderiza los componentes templ sin base de datos
     (panel de servicios, lista editable y de solo lectura, detalle,
     formulario con datos conservados, calendario, meses, escape de HTML).
2. **Pruebas Hurl** (`hurl/`): navegan la aplicación como lo haría el
   navegador (Hurl guarda las cookies de sesión entre pedidos) y verifican
   redirecciones, el HTML con consultas XPath y los códigos de error.

| Archivo | Qué prueba |
|---|---|
| `publico.hurl` | Portada sin JavaScript, panel con las 10 categorías (incluidas Hogar y Seguros) y sus logos, login y registro, redirección a `/login` sin sesión, `Cache-Control` del CSS. |
| `cuentas.hurl` | Registro con aviso de bienvenida, *"Hola, {nombre}"*, logout, login correcto e incorrecto y validaciones. |
| `suscripcion.hurl` | Alta (y validaciones), varias cuentas por servicio, detalle, ver la contraseña, modificar monto/usuario/contraseña, pausar desde *Modificar* (sin estado Cancelada), buscador, renovación automática, meses anteriores de solo lectura, eliminar y suscripciones ajenas. |
| `datos_de_prueba.hurl` | Que los usuarios de prueba del `.env` puedan entrar y vean sus suscripciones de ejemplo del mes (incluida la renovación automática). |

# Estructura del proyecto

```
.
├── main.go                 # Conexión, catálogo, renovación y servidor HTTP
├── main_test.go            # Tests de integración de sqlc
├── handlers/
│   ├── web.go              # Rutas, sesión y handlers HTML
│   └── sesiones.go         # Sesiones con cookie HttpOnly
├── logic/                  # Reglas de negocio (+ renovacion.go y sus tests, datos_de_prueba.go)
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
