# Requisitos previos

- [Go](https://go.dev/doc/install) 1.25 
- [Docker](https://docs.docker.com/get-docker/) 
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)
- [Hurl](https://hurl.dev/docs/installation.html)

# Ejecución

## Clonar repositorio
```bash
git clone https://github.com/nicolaskrohnbaldini14-rgb/Tp1-Krohn-Trapani-Cicopiedi.git
```
## Posicionarse en carpeta del proyecto
```bash
cd Tp1-Krohn-Trapani-Cicopiedi
```
## Cambiar a rama requerida
```bash
git checkout tp4
```
## Levantar la aplicación (TP4)
```bash
make run-app
```

`make run-app` realiza todas las tareas necesarias:

1. **Tareas previas:** borra contenedores y volúmenes anteriores, levanta PostgreSQL
   en Docker y espera a que el healthcheck indique que la base está lista, ejecuta
   `sqlc generate` y compila el ejecutable en `tmp/`.
2. **Ejecución:** deja corriendo el servidor en <http://localhost:8080>, que sirve
   tanto la API REST como el frontend.
3. **Tareas posteriores:** al detenerlo con `Ctrl+C` borra los contenedores y volúmenes.

Al iniciar, la aplicación precarga el catálogo de servicios (ver
[Catálogo de servicios](#catálogo-de-servicios)). Para probarla: registrarse,
crear suscripciones eligiendo un servicio del catálogo y pausarlas o
reactivarlas desde *Modificar*.

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

## Ejecutar pruebas (tests de sqlc + pruebas Hurl de la API)
```bash
make
```

# Frontend (TP4)

La interfaz está en [`static/`](static/) y la sirve el mismo servidor Go
(`http.FileServer` registrado en `/`), por lo que el navegador y la API
comparten origen: no hace falta configurar CORS y la cookie de sesión viaja
sola en cada `fetch`.

La aplicación está separada en páginas según quién la usa:

| Página | Quién | Contenido |
|---|---|---|
| `index.html` (`/`) | Todos | Portada centrada con el título, el lema y los botones **Iniciar sesión** y **Registrarse**, sobre el fondo azul. |
| `login.html` | Todos | Inicio de sesión. |
| `registro.html` | Todos | Registro de usuarios. |
| `cuenta.html` | Con sesión | *"Hola, {nombre}"*, resumen de lo que vence este mes y dos opciones: ver las suscripciones o crear una nueva. |
| `suscripciones.html` | Con sesión | Suscripciones que **vencen en el mes** (por defecto el actual) con el gasto del mes, un **buscador**, el botón *Modificar* (monto, cuenta y estado: activa o pausada) y el botón para eliminar. Abajo, links al mes anterior y al siguiente (`?mes=AAAA-MM`); al costado, un calendario del mes con los días de vencimiento marcados. Los **meses anteriores son solo de lectura**. |
| `nueva-suscripcion.html` | Con sesión | Formulario de alta: servicio del catálogo (agrupado por categoría; con `?servicio={id}` viene preseleccionado), monto y fechas; usuario y contraseña de la cuenta son opcionales (salvo para diferenciar una segunda suscripción al mismo servicio). La suscripción se crea siempre **Activa**. |

Todas las páginas tienen en la barra superior el botón **Servicios ▾**: al
pasar el cursor (o tocarlo en el celular) se despliega un panel **justo debajo
del botón**, con las categorías del catálogo una debajo de la otra (cada una
con su ícono, dibujado en SVG en `app.js`). Al hacer clic en una categoría se
despliegan sus servicios y empujan a las demás hacia abajo (acordeón: una
abierta por vez). Los mismos íconos aparecen al lado de cada
suscripción en *Mis suscripciones*. Lo completa `app.js` con
`GET /servicios`. Con la sesión iniciada, cada servicio es un link a
`nueva-suscripcion.html?servicio={id}`, que abre el formulario con ese
servicio ya elegido. El panel se cierra al sacar el cursor, con la × o con
Escape.

En el formulario de nueva suscripción, el selector de servicio abre el
catálogo en grande, centrado sobre la pantalla desenfocada (en vez de la
lista desplegable del navegador), con las categorías en grilla: al hacer clic
en una se despliegan sus servicios encima del resto, con el servicio actual
resaltado. Al tocar uno, queda elegido en el formulario y el panel se
cierra.

Con la sesión iniciada, la barra muestra las opciones como "pastillas" (la
página actual resaltada), separadas del usuario (inicial en un círculo y
*Hola, {nombre}*) y del botón *Cerrar sesión*. La barra va de margen a margen
de la pantalla. Los avisos (de éxito o de error) se muestran con fondo negro;
los de error llevan además una marca roja a la izquierda.

**Fondo:** todas las pantallas tienen sobre el azul una textura de grilla
suave (un SVG incrustado en el CSS).

**Caché del navegador:** el servidor envía los archivos de `static/` con
`Cache-Control: no-cache`, así el navegador siempre consulta si hay una
versión nueva. Además, cada página tiene `<meta name="version-pagina">`: si
`app.js` detecta una copia vieja guardada (con otra versión o sin la marca),
recarga la página una sola vez para traer la actual. Al cambiar el HTML hay
que actualizar esa versión en las páginas y en `VERSION_PAGINAS` de `app.js`.

Todas las páginas comparten [`static/app.js`](static/app.js) y
[`static/styles.css`](static/styles.css). Cada una se identifica con
`<body data-pagina="...">` y al cargar `app.js` ejecuta la función de esa página.

Funcionamiento de `app.js`:

- **Sesión:** al cargar cualquier página pide `GET /sesion`. Según la
  respuesta arma la barra de navegación (*Servicios ▾ / Iniciar sesión /
  Registrarse* o *Servicios ▾ / … / Hola, {nombre} / Cerrar sesión*). Las páginas privadas redirigen al login
  si no hay sesión.
- **Carga de datos:** cada página hace su `fetch` al cargar
  (`GET /servicios`, `GET /suscripciones/usuario/{id}`) y genera
  dinámicamente un `<li>` por entidad, con un botón **Eliminar** donde corresponde.
- **Alta:** los formularios tienen un listener de `submit` que hace
  `preventDefault()`, arma el JSON con los datos de los inputs y envía un
  `POST`. Al crear una suscripción se pasa a la lista de suscripciones, que
  vuelve a pedir la lista completa.
- **Vencimientos por mes:** `suscripciones.html` pide todas las suscripciones
  del usuario y muestra solo las que vencen en el mes elegido (por defecto el
  actual, o el de `?mes=AAAA-MM`). Arriba de la lista se muestra el gasto:
  - en el **mes actual** (y los siguientes), *Gasto mensual (activo)*: solo
    lo que **falta pagar**, es decir, las suscripciones activas que todavía
    no vencieron (lo que ya venció este mes no se suma);
  - en un **mes anterior**, *Gasto mensual*: todo lo que venció ese mes.

  Las pausadas no suman nunca. Debajo de la lista hay links al mes
  anterior y al siguiente, que muestran lo mismo para ese mes.
- **Calendario:** al costado de la lista se dibuja la grilla del mes (de lunes
  a domingo, con el día de hoy recuadrado). Los días con vencimientos tienen un
  pin y, al pasar el cursor o enfocarlos con el teclado, muestran un cartel con
  los servicios que vencen ese día, su monto y si están pausados.
- **Detalle de una suscripción:** en la lista solo se ve el servicio, su
  categoría, el monto y el estado. Al tocar una suscripción se abre un `<dialog>` (con el
  fondo desenfocado y efecto de zoom) con la cuenta, la fecha de inicio, el
  vencimiento y el estado. La contraseña de la cuenta aparece oculta: para
  verla hay que confirmar la contraseña del usuario de Suscripciones+
  (`POST /suscripciones/{id}/password`). Se cierra con la ×, con Escape o
  tocando el fondo, y al cerrarse la contraseña vuelve a ocultarse.
- **Buscador:** un campo con una lupa que filtra las suscripciones del mes
  que se está viendo por servicio, categoría o usuario de la cuenta, a medida
  que se escribe.
- **Solo lectura:** en los meses anteriores al actual, y en los períodos que
  ya se renovaron (*Vencida (renovada)*), no aparecen los botones de
  modificar ni de eliminar. La API también lo
  controla (`403`).
- **Modificar una suscripción:** cada suscripción editable tiene el botón
  *Modificar* en la lista (también está dentro del detalle), que abre un
  formulario con el monto, el usuario y la contraseña de la cuenta del
  servicio y el **estado** (*Activa* o *Pausada*). Si la contraseña se deja
  vacía se conserva la guardada; si se borra el usuario, queda "Sin cuenta".
  Hace un `PUT` y refresca la lista y el total (las pausadas no suman). No
  existe el estado *Cancelada*: para dejar una suscripción se la elimina.
- **Borrado con delegación de eventos:** como los botones se crean
  dinámicamente, hay un único listener de `click` sobre cada lista que detecta
  el botón presionado, toma el ID desde `data-id` y hace el `DELETE`. Tras un
  borrado exitoso se refresca la lista.
- **Errores:** los mensajes de la API (email repetido, servicio con
  suscripciones, fechas inválidas, etc.) se muestran en un aviso arriba de la
  página, y el formulario conserva lo que se había escrito.
- **Seguridad:** los datos se insertan con `textContent`, nunca con
  `innerHTML`, para no inyectar HTML ingresado por los usuarios. Además, los
  permisos los controla la API: ocultar un botón en la página no alcanza.

Los estilos usan **Flexbox** (barra, formularios y listas) y **CSS Grid**
(opciones de la cuenta y formulario de alta), con la paleta de la
primera entrega: azul acero `#567c8d`, azul petróleo `#002f44`/`#004b6e` y
crema `#ece6d4`.



# Documentación de persistencia

El modelo de datos está compuesto por 4 tablas relacionales, definidas en
[`db/schema/schema.sql`](db/schema/schema.sql):

| Tabla | Descripción |
|---|---|
| `usuario` | Cuenta de la persona que usa la app (nombre, email y contraseña hasheada con bcrypt). |
| `servicio` | Catálogo de servicios disponibles (nombre + categoría), normalizado para poder agrupar el gasto por categoría. Se precarga al iniciar la app. |
| `suscripcion` | Relaciona un `usuario` con un `servicio`: monto, fecha de inicio, fecha de vencimiento, credenciales de acceso al servicio (usuario/contraseña de esa cuenta, **opcionales**: pueden ser `NULL`) y estado (activa o pausada). |
| `pago` | Historial real de cobros de una suscripción. Permite calcular el gasto efectivo mes a mes, no solo el proyectado. |

**Relaciones:**

<img width="1024" height="559" alt="ModeloRelacionalWEB" src="https://github.com/user-attachments/assets/a5284446-98ce-4dc8-b4a0-a3bec07d5b3e" />

- `suscripcion.id_usuario` → `usuario.id_usuario` (`ON DELETE CASCADE`)
- `suscripcion.id_servicio` → `servicio.id_servicio`
- `pago.id_suscripcion` → `suscripcion.id_suscripcion` (`ON DELETE CASCADE`)

El acceso a datos se genera con [sqlc](https://sqlc.dev/) a partir de las
queries en [`db/queries/queries.sql`](db/queries/queries.sql), que cubren las
operaciones CRUD (Create, Get, List, Update, Delete) para cada una de las 4
entidades. El código generado vive en `db/sqlc/` (`models.go`, `queries.sql.go`,
`db.go`) y **no debe editarse a mano** — se regenera automáticamente cada vez
que se corre `sqlc generate` (parte de `make run-app` y `make test-api`).

Los tests de integración (`main_test.go`) validan el CRUD contra una base
Postgres real levantada en Docker. Hay un test independiente por tabla
(`TestUsuario_CRUD`, `TestServicio_CRUD`, `TestSubscripciones_CRUD` y
`TestPago_CRUD`), y cada uno arranca con las tablas vacías y crea los datos
que necesita con funciones auxiliares compartidas (por ejemplo, el test de
pagos crea primero un usuario, un servicio y una suscripción). Se puede correr
uno solo con `go test -v -run TestPago_CRUD .`

# API REST

La lógica de negocio y los handlers HTTP de cada entidad están en `logic/`, y
las rutas se registran en `main.go`. Todas las respuestas exitosas con cuerpo
son JSON. Los errores devuelven un mensaje de texto con el código HTTP
correspondiente.

### Sesión y permisos

| Método | Ruta | Respuestas |
|---|---|---|
| `POST` | `/sesion` | `200` usuario + cookie de sesión · `401` email o contraseña incorrectos |
| `GET` | `/sesion` | `200` usuario de la sesión · `401` sin sesión |
| `DELETE` | `/sesion` | `204` cierra la sesión |

```json
{ "email": "nico@example.com", "password": "secreto" }
```

- La sesión es un token aleatorio guardado en memoria del servidor y enviado
  en una cookie `HttpOnly` y `SameSite=Lax` llamada `sesion` (dura 24 h). Si
  se reinicia el servidor hay que volver a ingresar.
- Las contraseñas se guardan **hasheadas con bcrypt** (`golang.org/x/crypto/bcrypt`).
- `logic/sesion.go` tiene el middleware `Proteger`, que en `main.go` se aplica
  solo a los métodos que lo necesitan:

| Recurso | Público | Requiere sesión |
|---|---|---|
| Usuarios | `GET`, `POST /usuarios` (registro) | `PUT`/`DELETE`: solo la propia cuenta (`403` si es otra) |
| Servicios | `GET` (catálogo precargado) | — (no se pueden crear, modificar ni borrar: `405`) |
| Suscripciones | — | Se crean a nombre del usuario de la sesión y cada uno ve y modifica solo las suyas (`404` si son de otro) |
| Pagos | — | Solo sobre las propias suscripciones (`404` si son de otro) |

Sin sesión, las rutas protegidas responden `401`.

### Usuarios

| Método | Ruta | Respuestas |
|---|---|---|
| `POST` | `/usuarios` | `201` creado · `400` datos inválidos · `409` email ya registrado |
| `GET` | `/usuarios` | `200` lista de usuarios |
| `GET` | `/usuarios/{id}` | `200` · `400` ID no numérico · `404` no encontrado |
| `PUT` | `/usuarios/{id}` | `200` · `400` · `404` · `409` email en uso |
| `DELETE` | `/usuarios/{id}` | `204` · `404` |

```json
{ "nombre": "Nico", "email": "nico@example.com", "password": "secreto" }
```

### Servicios

| Método | Ruta | Respuestas |
|---|---|---|
| `GET` | `/servicios` | `200` catálogo completo |
| `GET` | `/servicios/{id}` | `200` · `400` ID no numérico · `404` no encontrado |
| `POST` / `PUT` / `PATCH` / `DELETE` | `/servicios`, `/servicios/{id}` | `405` el catálogo viene precargado |

### Catálogo de servicios

Al iniciar, `main.go` llama a `CargarCatalogoInicial` (`logic/servicio.go`),
que inserta con la query `CrearServicioSiNoExiste`
(`INSERT ... ON CONFLICT (nombre) DO NOTHING`) los servicios que todavía no
existan. Se puede ejecutar en cada arranque sin duplicar nada.

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

La categoría que antes era *Gimnasio* ahora es **Deportes**, para poder
registrar gimnasio, pádel, fútbol, etc.

### Suscripciones

| Método | Ruta | Respuestas |
|---|---|---|
| `POST` | `/suscripciones` | `201` · `400` datos inválidos o falta el usuario de cuenta para diferenciarla · `403` a nombre de otro usuario · `404` servicio inexistente · `409` ya tiene una suscripción vigente a ese servicio con esa cuenta |
| `GET` | `/suscripciones/{id}` | `200` · `400` · `404` |
| `GET` | `/suscripciones/usuario/{id_usuario}` | `200` suscripciones del usuario · `403` si es otro usuario |
| `PUT` | `/suscripciones/{id}` | `200` · `400` · `403` vencida o de un mes anterior · `404` · `409` cuenta repetida |
| `DELETE` | `/suscripciones/{id}` | `204` · `403` vencida o de un mes anterior · `404` |
| `POST` | `/suscripciones/{id}/password` | `200` `{"password_cuenta": "..."}` (o `null` si no tiene) · `403` contraseña del usuario incorrecta · `404` si no existe o es de otro usuario |

- `id_usuario` es opcional: si no se envía se usa el de la sesión.
- `usuario_cuenta` y `password_cuenta` son opcionales (por ejemplo, una
  cancha de pádel no tiene cuenta); si no se envían o vienen vacíos quedan en
  `null`. En un `PUT`, si no se envía `password_cuenta` se conserva la guardada.
- `password_cuenta` nunca viene en las respuestas. Para verla se usa
  `POST /suscripciones/{id}/password` con `{"password": "<contraseña del usuario>"}`:
  se confirma con bcrypt que la contraseña sea la del usuario de la sesión.
- `estado` es opcional al crear y, si se envía, tiene que ser `Activa`:
  pausar o cancelar es un cambio posterior (`PUT`).

```json
{
  "id_servicio": 1,
  "monto": "2500.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "nico@example.com",
  "password_cuenta": "passcuenta123"
}
```

### Pagos

| Método | Ruta | Respuestas |
|---|---|---|
| `POST` | `/pagos` | `201` · `400` datos inválidos o fecha fuera de vigencia · `404` suscripción inexistente |
| `GET` | `/pagos/{id}` | `200` · `400` · `404` |
| `GET` | `/pagos/suscripcion/{id_suscripcion}` | `200` historial de pagos · `404` suscripción inexistente |
| `DELETE` | `/pagos/{id}` | `204` · `404` |
| `PUT` / `PATCH` | `/pagos/{id}` | `405` los pagos son un historial y no se modifican |

```json
{ "id_suscripcion": 1, "monto": "2500.00", "fecha_pago": "2026-06-15T00:00:00Z" }
```

# Reglas de negocio

- **Usuarios:** nombre, email y password son obligatorios. El email tiene que tener un formato válido y no puede repetirse. La password se guarda hasheada y nunca se devuelve en las respuestas.
- **Acceso:** cada usuario ve y modifica únicamente sus propias suscripciones y pagos.
- **Servicios:** el catálogo viene precargado y es de solo lectura. Las categorías son `Streaming`, `Música`, `Software`, `Hogar`, `Seguros`, `Deportes`, `Salud`, `Financiero`, `Turismo` y `Otros`.
- **Suscripciones:**
  - El usuario y el servicio tienen que existir.
  - El monto tiene que ser mayor a cero y la fecha de inicio anterior a la de vencimiento.
  - Se crean siempre en estado `Activa`; después se pueden pausar y reactivar (`Activa` y `Pausada` son los únicos estados que se pueden elegir). No hay estado `Cancelada`: para dejar una suscripción se la elimina.
  - **Renovación automática:** cuando una suscripción activa vence, se renueva sola por el mismo plazo con el que se registró (si era de un mes, por otro mes; si era de un año, por otro año). El período vencido queda como historial con estado `Vencida` y se crea uno nuevo con el mismo monto y la misma cuenta. Se hace al iniciar la app y cada vez que se piden las suscripciones de un usuario (`RenovarVencidas` en `logic/renovacion.go`), con una sola sentencia SQL atómica (`RenovarSuscripcion`). Las pausadas no se renuevan.
  - Las suscripciones `Vencida` y las que vencen en un mes anterior al actual son de **solo lectura**.
  - Usuario y contraseña de la cuenta del servicio son opcionales.
  - Un usuario puede tener **varias suscripciones al mismo servicio** si son cuentas distintas: desde la segunda (mientras la otra siga activa o pausada) hay que indicar el usuario de la cuenta, y no se puede repetir uno que ya esté en uso (sin distinguir mayúsculas). Esta regla está en `verificarCuentaDistinta`, en `logic/suscripcion.go`.
  - La `password_cuenta` nunca se devuelve en las respuestas.
- **Pagos:** la suscripción tiene que existir, el monto tiene que ser mayor a cero y la fecha de pago no puede ser posterior al vencimiento de la suscripción.

# Pruebas de los endpoints

Las pruebas están en `hurl/`. Hurl guarda las cookies entre pedidos del mismo
archivo, así que cada archivo inicia sesión con la cuenta que necesita antes
de las acciones protegidas.

| Archivo | Qué prueba |
|---|---|
| `frontend.hurl` | Que se sirvan las páginas, que el inicio tenga los links a login y registro y el menú de servicios en la barra, y que no tenga el alta de servicios. |
| `usuario.hurl` | Registro, login correcto e incorrecto, `GET /sesion`, CRUD de la propia cuenta, `403` sobre cuentas ajenas y cierre de sesión. |
| `servicio.hurl` | Catálogo precargado (servicios y categorías, incluida Deportes) y `405` al intentar crear, modificar o borrar servicios. |
| `suscripcion.hurl` | CRUD de suscripciones, varias suscripciones al mismo servicio con cuentas distintas, alta sin credenciales, `400` al crearlas pausadas o con el estado `Cancelada` (ya no existe), pausar y reactivar, renovación automática con el mismo plazo, `403` al modificar períodos vencidos, `404` sobre suscripciones ajenas y `401` sin sesión. |
| `pago.hurl` | Pagos sobre suscripciones propias, `404` sobre pagos ajenos y `401` sin sesión. |
| `datos_de_prueba.hurl` | Que los usuarios de prueba del `.env` puedan entrar y tengan sus suscripciones de ejemplo (incluida la renovación automática). |

# Estructura del proyecto

```
.
├── main.go               # Conexión a la base, armado de capas, rutas HTTP y frontend
├── main_test.go          # Tests de integración CRUD de cada tabla (sqlc)
├── static/               # Frontend del TP4 (servido en /)
│   ├── index.html        # Inicio público
│   ├── login.html
│   ├── registro.html
│   ├── cuenta.html       # "Hola, {nombre}" (consumidores)
│   ├── suscripciones.html
│   ├── nueva-suscripcion.html
│   ├── app.js            # Lógica de todas las páginas
│   └── styles.css
├── logic/                # Lógica de negocio y handlers HTTP de cada entidad
│   ├── sesion.go         # Sesiones, login/logout y middleware de permisos
│   ├── usuario.go
│   ├── servicio.go
│   ├── suscripcion.go    # Alta, modificación y baja (incluye la regla de cuentas distintas)
│   ├── renovacion.go     # Renovación automática y períodos de solo lectura
│   ├── pago.go
│   └── datos_de_prueba.go # Usuarios de prueba del .env con suscripciones de ejemplo
├── db/
│   ├── schema/schema.sql # Tablas y claves foráneas
│   ├── queries/queries.sql
│   └── sqlc/             # Código generado por sqlc (no se versiona)
├── hurl/                 # Pruebas de los endpoints
│   ├── frontend.hurl
│   ├── pago.hurl
│   ├── servicio.hurl
│   ├── suscripcion.hurl
│   ├── usuario.hurl
│   └── datos_de_prueba.hurl
├── docker-compose.yml    # PostgreSQL con healthcheck (credenciales del .env)
├── .env.example          # Variables de entorno de ejemplo (el .env no se versiona)
├── .gitignore
├── sqlc.yaml
└── makefile
```
