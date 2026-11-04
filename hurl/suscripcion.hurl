# Mis suscripciones: alta, detalle, modificar, contraseña, estados, renovación
# automática, meses anteriores de solo lectura y suscripciones ajenas.

# ---------- Setup: consumidor con sesión iniciada ----------
POST {{host}}/registro
[FormParams]
nombre: Usuario Subs
email: usuario.subs@example.com
password: secreto123
HTTP 303

# los ids de los servicios se toman del formulario (agrupado por categoría)
GET {{host}}/cuenta/suscripciones/nueva
HTTP 200
[Captures]
netflix_id: xpath "string(//select[@name='id_servicio']//option[normalize-space(.)='Netflix']/@value)"
padel_id: xpath "string(//select[@name='id_servicio']//option[normalize-space(.)='Pádel']/@value)"
spotify_id: xpath "string(//select[@name='id_servicio']//option[normalize-space(.)='Spotify']/@value)"
[Asserts]
xpath "count(//select[@name='id_servicio']/optgroup)" == 10
# se crea siempre activa y el usuario sale de la sesión
xpath "count(//select[@name='estado'])" == 0
xpath "count(//select[@name='id_usuario'])" == 0


# ---------- 1. Crear suscripción: redirige al mes en que vence (303) ----------
POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: {{netflix_id}}
monto: 1500
fecha_inicio: 2026-01-01
fecha_vencimiento: 2026-12-31
usuario_cuenta: cuenta@test.com
password_cuenta: clavecuenta
HTTP 303
[Asserts]
header "Location" == "/cuenta/suscripciones?mes=2026-12&ok=creada"


# ---------- 2. Aparece en la lista del mes, sin datos de la cuenta ----------
GET {{host}}/cuenta/suscripciones?mes=2026-12&ok=creada
HTTP 200
[Captures]
sub_id: xpath "string(//li[.//strong[starts-with(normalize-space(.),'Netflix — $ 1.500,00')]]//a[@class='item-enlace']/@href)" regex "ver=(\\d+)"
[Asserts]
xpath "normalize-space(//div[contains(@class,'mensaje')])" == "Suscripción creada."
xpath "count(//li[contains(@class,'item')]//span[@class='icono-categoria'])" == 1
xpath "normalize-space(//p[@class='total'])" == "Gasto mensual activo de diciembre de 2026: $ 1.500,00"
body not contains "cuenta@test.com"
body not contains "clavecuenta"


# ---------- 3. Varias suscripciones al mismo servicio: cuentas distintas ----------
POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: {{netflix_id}}
monto: 900
fecha_inicio: 2026-01-01
fecha_vencimiento: 2026-12-31
usuario_cuenta: otra@test.com
HTTP 303

# la misma cuenta (sin importar mayúsculas) no se puede repetir (409)
POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: {{netflix_id}}
monto: 900
fecha_inicio: 2026-01-01
fecha_vencimiento: 2026-12-31
usuario_cuenta: CUENTA@test.com
HTTP 409
[Asserts]
xpath "normalize-space(//div[contains(@class,'mensaje')])" contains "con la cuenta"

# desde la segunda hay que indicar el usuario de la cuenta (400)
POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: {{netflix_id}}
monto: 900
fecha_inicio: 2026-01-01
fecha_vencimiento: 2026-12-31
HTTP 400
[Asserts]
xpath "normalize-space(//div[contains(@class,'mensaje')])" contains "indicá el usuario de la cuenta"

# sin credenciales: se permite (por ejemplo, una cancha de pádel)
POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: {{padel_id}}
monto: 15000
fecha_inicio: 2026-01-01
fecha_vencimiento: 2026-12-15
HTTP 303

GET {{host}}/cuenta/suscripciones?mes=2026-12
HTTP 200
[Captures]
sub_otra_id: xpath "string(//li[.//strong[starts-with(normalize-space(.),'Netflix — $ 900,00')]]//a[@class='item-enlace']/@href)" regex "ver=(\\d+)"
[Asserts]
xpath "count(//ul[@id='lista-suscripciones']/li[contains(@class,'item')])" == 3
xpath "normalize-space(//p[@class='total'])" == "Gasto mensual activo de diciembre de 2026: $ 17.400,00"
# el calendario marca los días 15 y 31
xpath "count(//div[contains(@class,'cal-con-vencimiento')])" == 2


# ---------- 4. Validaciones del alta (se conservan los datos) ----------
POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: {{spotify_id}}
monto: 0
fecha_inicio: 2026-01-01
fecha_vencimiento: 2026-12-31
HTTP 400
[Asserts]
xpath "normalize-space(//div[contains(@class,'mensaje')])" contains "mayor a cero"
xpath "string(//input[@name='fecha_inicio']/@value)" == "2026-01-01"
xpath "string(//select[@name='id_servicio']//option[@selected]/@value)" == "{{spotify_id}}"

POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: {{spotify_id}}
monto: 100
fecha_inicio: 2026-12-31
fecha_vencimiento: 2026-01-01
HTTP 400
[Asserts]
xpath "normalize-space(//div[contains(@class,'mensaje')])" contains "anterior"

POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: {{spotify_id}}
monto: 100
fecha_inicio:
fecha_vencimiento:
HTTP 400
[Asserts]
xpath "normalize-space(//div[contains(@class,'mensaje')])" contains "obligatorias"

POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: 999999
monto: 100
fecha_inicio: 2026-01-01
fecha_vencimiento: 2026-12-31
HTTP 404
[Asserts]
xpath "normalize-space(//div[contains(@class,'mensaje')])" contains "no existe"


# ---------- 5. Detalle: cuenta, fechas y contraseña oculta ----------
GET {{host}}/cuenta/suscripciones?mes=2026-12&ver={{sub_id}}
HTTP 200
[Asserts]
xpath "count(//div[@class='detalle-fondo'])" == 1
xpath "normalize-space(//article[@id='detalle']//h2)" == "Netflix"
xpath "normalize-space(//dl[@class='detalle-datos']/dd[2])" == "cuenta@test.com"
xpath "normalize-space(//dl[@class='detalle-datos']/dd[3])" == "01/01/2026"
xpath "normalize-space(//dl[@class='detalle-datos']/dd[4])" == "31/12/2026"
xpath "normalize-space(//dl[@class='detalle-datos']/dd[5])" == "••••••••"
body not contains "clavecuenta"


# ---------- 6. Ver la contraseña confirmando la del usuario ----------
POST {{host}}/cuenta/suscripciones/{{sub_id}}/password
[FormParams]
mes: 2026-12
password: incorrecta
HTTP 403
[Asserts]
xpath "normalize-space(//p[@class='detalle-error'])" contains "no es correcta"
body not contains "clavecuenta"

POST {{host}}/cuenta/suscripciones/{{sub_id}}/password
[FormParams]
mes: 2026-12
password: secreto123
HTTP 200
[Asserts]
xpath "normalize-space(//span[@class='revelada'])" == "clavecuenta"


# ---------- 7. Modificar monto, usuario y contraseña de la cuenta ----------
GET {{host}}/cuenta/suscripciones?mes=2026-12&ver={{sub_id}}&editar=1
HTTP 200
[Asserts]
xpath "string(//form[contains(@action,'/modificar')]//input[@name='monto']/@value)" == "1500.00"
xpath "string(//form[contains(@action,'/modificar')]//input[@name='usuario_cuenta']/@value)" == "cuenta@test.com"
xpath "string(//form[contains(@action,'/modificar')]//input[@name='password_cuenta']/@value)" == ""
# el estado se elige en el mismo formulario: solo Activa o Pausada
xpath "count(//form[contains(@action,'/modificar')]//select[@name='estado']/option)" == 2
xpath "string(//form[contains(@action,'/modificar')]//select[@name='estado']/option[@selected]/@value)" == "Activa"

POST {{host}}/cuenta/suscripciones/{{sub_id}}/modificar
[FormParams]
mes: 2026-12
monto: 1999.50
usuario_cuenta: cambiada@test.com
password_cuenta: otraclave
HTTP 303
[Asserts]
header "Location" == "/cuenta/suscripciones?mes=2026-12&ver={{sub_id}}&ok=modificada"

# sin escribir contraseña nueva se conserva la guardada
POST {{host}}/cuenta/suscripciones/{{sub_id}}/modificar
[FormParams]
mes: 2026-12
monto: 2000
usuario_cuenta: cambiada@test.com
password_cuenta:
HTTP 303

POST {{host}}/cuenta/suscripciones/{{sub_id}}/password
[FormParams]
mes: 2026-12
password: secreto123
HTTP 200
[Asserts]
xpath "normalize-space(//span[@class='revelada'])" == "otraclave"
xpath "normalize-space(//dl[@class='detalle-datos']/dd[2])" == "cambiada@test.com"
xpath "normalize-space(//p[@class='detalle-monto'])" == "$ 2.000,00 por mes"

# una cuenta que ya usa otra suscripción al mismo servicio (409)
POST {{host}}/cuenta/suscripciones/{{sub_id}}/modificar
[FormParams]
mes: 2026-12
monto: 2000
usuario_cuenta: otra@test.com
HTTP 409
[Asserts]
xpath "normalize-space(//p[@class='detalle-error'])" contains "con la cuenta"
# el formulario conserva lo escrito
xpath "string(//form[contains(@action,'/modificar')]//input[@name='usuario_cuenta']/@value)" == "otra@test.com"


# ---------- 8. Pausar desde "Modificar" ----------
# en la lista ya no hay selector de estado: solo se muestra
GET {{host}}/cuenta/suscripciones?mes=2026-12
HTTP 200
[Asserts]
xpath "count(//ul[@id='lista-suscripciones']//select[@name='estado'])" == 0
xpath "normalize-space(//li[.//a[contains(@href,'ver={{sub_id}}')]]//span[@class='estado-fijo'])" == "Activa"

POST {{host}}/cuenta/suscripciones/{{sub_id}}/modificar
[FormParams]
mes: 2026-12
monto: 2000
usuario_cuenta: cambiada@test.com
estado: Pausada
HTTP 303
[Asserts]
header "Location" == "/cuenta/suscripciones?mes=2026-12&ver={{sub_id}}&ok=modificada"

GET {{host}}/cuenta/suscripciones?mes=2026-12
HTTP 200
[Asserts]
xpath "normalize-space(//li[.//a[contains(@href,'ver={{sub_id}}')]]//span[@class='estado-fijo'])" == "Pausada"
# la pausada no cuenta para el gasto
xpath "normalize-space(//p[@class='total'])" == "Gasto mensual activo de diciembre de 2026: $ 15.900,00"

# al volver a modificar, el formulario muestra el estado actual
GET {{host}}/cuenta/suscripciones?mes=2026-12&ver={{sub_id}}&editar=1
HTTP 200
[Asserts]
xpath "string(//form[contains(@action,'/modificar')]//select[@name='estado']/option[@selected]/@value)" == "Pausada"

# no se puede pasar a Vencida a mano ni cancelar: para dejarla se elimina (400)
POST {{host}}/cuenta/suscripciones/{{sub_id}}/modificar
[FormParams]
mes: 2026-12
monto: 2000
estado: Vencida
HTTP 400

POST {{host}}/cuenta/suscripciones/{{sub_id}}/modificar
[FormParams]
mes: 2026-12
monto: 2000
estado: Cancelada
HTTP 400
[Asserts]
xpath "normalize-space(//p[@class='detalle-error'])" contains "no es válido"

# la ruta vieja para cambiar el estado ya no existe
POST {{host}}/cuenta/suscripciones/{{sub_id}}/estado
[FormParams]
estado: Activa
HTTP 404


# ---------- 9. Buscador ----------
GET {{host}}/cuenta/suscripciones?mes=2026-12&q=deportes
HTTP 200
[Asserts]
xpath "count(//ul[@id='lista-suscripciones']/li[contains(@class,'item')])" == 1
xpath "normalize-space(//ul[@id='lista-suscripciones']//strong)" startsWith "Pádel"

GET {{host}}/cuenta/suscripciones?mes=2026-12&q=inexistente
HTTP 200
[Asserts]
xpath "normalize-space(//li[@class='vacio'])" contains "coincide con"


# ---------- 10. Renovación automática y meses anteriores de solo lectura ----------
# una suscripción anual que venció en marzo de 2025: se renueva sola por el
# mismo plazo (un año) hasta llegar a un período vigente
POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: {{spotify_id}}
monto: 2900
fecha_inicio: 2024-03-10
fecha_vencimiento: 2025-03-10
usuario_cuenta: renueva@test.com
HTTP 303
[Asserts]
header "Location" == "/cuenta/suscripciones?mes=2025-03&ok=creada"

GET {{host}}/cuenta/suscripciones?mes=2025-03
HTTP 200
[Captures]
sub_vieja_id: xpath "string(//li[.//strong[starts-with(normalize-space(.),'Spotify')]]//a[@class='item-enlace']/@href)" regex "ver=(\\d+)"
[Asserts]
xpath "count(//p[@class='aviso-lectura'])" == 1
xpath "normalize-space(//li[contains(@class,'item-solo-lectura')]//span[@class='estado-fijo'])" == "Vencida (renovada)"
# en un mes anterior no hay botones para modificar ni eliminar
xpath "count(//a[@class='btn-modificar'])" == 0
xpath "count(//form[contains(@action,'/eliminar')])" == 0

# el período siguiente (marzo de 2026) mantiene el día y el monto
GET {{host}}/cuenta/suscripciones?mes=2026-03
HTTP 200
[Asserts]
xpath "normalize-space(//li[contains(@class,'item')]//strong)" == "Spotify — $ 2.900,00"
xpath "count(//div[contains(@class,'cal-con-vencimiento')]//span[normalize-space(.)='10'])" == 1

# y existe el de marzo de 2027
GET {{host}}/cuenta/suscripciones?mes=2027-03
HTTP 200
[Asserts]
xpath "count(//li[contains(@class,'item')]//strong[starts-with(normalize-space(.),'Spotify')])" == 1

# el período vencido no se puede modificar, cambiar de estado ni borrar (403)
POST {{host}}/cuenta/suscripciones/{{sub_vieja_id}}/eliminar
[FormParams]
mes: 2025-03
HTTP 403
[Asserts]
xpath "normalize-space(//div[contains(@class,'mensaje')])" contains "no se pueden modificar"

POST {{host}}/cuenta/suscripciones/{{sub_vieja_id}}/modificar
[FormParams]
mes: 2025-03
monto: 1
HTTP 403

POST {{host}}/cuenta/suscripciones/{{sub_vieja_id}}/modificar
[FormParams]
mes: 2025-03
monto: 2900
estado: Pausada
HTTP 403

# el detalle se puede ver, pero sin "Modificar"
GET {{host}}/cuenta/suscripciones?mes=2025-03&ver={{sub_vieja_id}}&editar=1
HTTP 200
[Asserts]
xpath "count(//article[@id='detalle']//p[@class='aviso-lectura'])" == 1
xpath "count(//form[contains(@action,'/modificar')])" == 0


# ---------- 11. Eliminar una suscripción propia (303) ----------
POST {{host}}/cuenta/suscripciones/{{sub_otra_id}}/eliminar
[FormParams]
mes: 2026-12
HTTP 303
[Asserts]
header "Location" == "/cuenta/suscripciones?mes=2026-12&ok=eliminada"

GET {{host}}/cuenta/suscripciones?mes=2026-12
HTTP 200
[Asserts]
xpath "count(//strong[starts-with(normalize-space(.),'Netflix — $ 900,00')])" == 0

POST {{host}}/logout
HTTP 303


# ---------- 12. Otro usuario no ve ni toca suscripciones ajenas (404) ----------
POST {{host}}/registro
[FormParams]
nombre: Otro Consumidor
email: otro.consumidor@example.com
password: secreto123
HTTP 303

GET {{host}}/cuenta/suscripciones?mes=2026-12&ver={{sub_id}}
HTTP 200
[Asserts]
xpath "count(//div[@class='detalle-fondo'])" == 0
xpath "count(//strong[starts-with(normalize-space(.),'Netflix')])" == 0

POST {{host}}/cuenta/suscripciones/{{sub_id}}/eliminar
[FormParams]
mes: 2026-12
HTTP 404

POST {{host}}/cuenta/suscripciones/{{sub_id}}/password
[FormParams]
mes: 2026-12
password: secreto123
HTTP 404
[Asserts]
body not contains "otraclave"

POST {{host}}/cuenta/suscripciones/{{sub_id}}/modificar
[FormParams]
mes: 2026-12
monto: 1
HTTP 404

POST {{host}}/logout
HTTP 303
