# ---------- Setup: servicios del catálogo precargado ----------
GET {{host}}/servicios
HTTP 200
[Captures]
servicio_id: jsonpath "$[?(@.nombre=='Netflix')].id_servicio" nth 0
padel_id: jsonpath "$[?(@.nombre=='Pádel')].id_servicio" nth 0


# sin sesión no se pueden crear suscripciones (401)
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_servicio": {{servicio_id}},
  "monto": "1500.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z"
}
HTTP 401


# ---------- Setup: consumidor con sesión iniciada ----------
POST {{host}}/usuarios
Content-Type: application/json
{
  "nombre": "Usuario Suscripciones",
  "email": "subs_{{newUuid}}@test.com",
  "password": "secreto123"
}
HTTP 201
[Captures]
usuario_id: jsonpath "$.id_usuario"
usuario_email: jsonpath "$.email"

# --- login como consumidor ---
POST {{host}}/sesion
Content-Type: application/json
{
  "email": "{{usuario_email}}",
  "password": "secreto123"
}
HTTP 200


# ---------- 1. Crear suscripcion (201) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": {{servicio_id}},
  "monto": "1500.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "cuenta@test.com",
  "password_cuenta": "clavecuenta",
  "estado": "Activa"
}
HTTP 201
[Captures]
sub_id: jsonpath "$.id_suscripcion"
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$.id_suscripcion" > 0
jsonpath "$.id_usuario" == {{usuario_id}}
jsonpath "$.id_servicio" == {{servicio_id}}
jsonpath "$.monto" == "1500.00"
jsonpath "$.estado" == "Activa"
jsonpath "$.usuario_cuenta" == "cuenta@test.com"
jsonpath "$.password_cuenta" == null
jsonpath "$.fecha_inicio" startsWith "2026-01-01"
jsonpath "$.fecha_vencimiento" startsWith "2026-12-31"


# ---------- 2. Varias suscripciones al mismo servicio: cuentas distintas ----------
# otra cuenta del mismo servicio: se permite (201)
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": {{servicio_id}},
  "monto": "900.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "otra@test.com",
  "password_cuenta": "clave"
}
HTTP 201
[Captures]
sub_otra_id: jsonpath "$.id_suscripcion"

# la misma cuenta (sin importar mayúsculas) no se puede repetir (409)
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_servicio": {{servicio_id}},
  "monto": "900.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "CUENTA@test.com"
}
HTTP 409
[Asserts]
body contains "con la cuenta"

# desde la segunda hay que indicar el usuario de la cuenta (400)
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_servicio": {{servicio_id}},
  "monto": "900.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z"
}
HTTP 400
[Asserts]
body contains "indicá el usuario de la cuenta"


# ---------- 3. A nombre de otro usuario (403) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": 999999,
  "id_servicio": {{servicio_id}},
  "monto": "100.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "x@test.com",
  "password_cuenta": "clave",
  "estado": "Activa"
}
HTTP 403
[Asserts]
body contains "a tu nombre"


# ---------- 4. Servicio inexistente (404) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": 999999,
  "monto": "100.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "x@test.com",
  "password_cuenta": "clave",
  "estado": "Activa"
}
HTTP 404
[Asserts]
body contains "servicio"


# ---------- 5. IDs invalidos (400) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": 0,
  "id_servicio": 0,
  "monto": "100.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "x@test.com",
  "password_cuenta": "clave",
  "estado": "Activa"
}
HTTP 400


# ---------- 6. Monto cero (400) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": {{servicio_id}},
  "monto": "0",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "x@test.com",
  "password_cuenta": "clave",
  "estado": "Activa"
}
HTTP 400
[Asserts]
body contains "ingrese un monto válido"

# un monto que no entra en la base (DECIMAL(10,2)) es un error del usuario
# (400), no un error interno (500)
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_servicio": {{servicio_id}},
  "monto": "99999999999",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z"
}
HTTP 400
[Asserts]
body contains "ingrese un monto válido"

# el máximo que entra sí se acepta
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_servicio": {{servicio_id}},
  "monto": "99999999.99",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "maximo@test.com"
}
HTTP 201
[Captures]
sub_maxima_id: jsonpath "$.id_suscripcion"
[Asserts]
jsonpath "$.monto" == "99999999.99"

DELETE {{host}}/suscripciones/{{sub_maxima_id}}
HTTP 204


# ---------- 7. Monto no numerico (400) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": {{servicio_id}},
  "monto": "abc",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "x@test.com",
  "password_cuenta": "clave",
  "estado": "Activa"
}
HTTP 400
[Asserts]
body contains "ingrese un monto válido"


# ---------- 8. Inicio posterior al vencimiento (400) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": {{servicio_id}},
  "monto": "100.00",
  "fecha_inicio": "2027-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-01-01T00:00:00Z",
  "usuario_cuenta": "x@test.com",
  "password_cuenta": "clave",
  "estado": "Activa"
}
HTTP 400
[Asserts]
body contains "anterior"


# ---------- 9. Sin fechas (400) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": {{servicio_id}},
  "monto": "100.00",
  "usuario_cuenta": "x@test.com",
  "password_cuenta": "clave",
  "estado": "Activa"
}
HTTP 400
[Asserts]
body contains "obligatorias"


# ---------- 10. Sin credenciales de cuenta (201): son opcionales ----------
# por ejemplo, una cancha de pádel no tiene usuario ni contraseña
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_servicio": {{padel_id}},
  "monto": "15000.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z"
}
HTTP 201
[Captures]
padel_sub_id: jsonpath "$.id_suscripcion"
[Asserts]
jsonpath "$.estado" == "Activa"
jsonpath "$.usuario_cuenta" == null
jsonpath "$.password_cuenta" == null

# al pedir la contraseña de una suscripción sin cuenta, viene null
POST {{host}}/suscripciones/{{padel_sub_id}}/password
Content-Type: application/json
{
  "password": "secreto123"
}
HTTP 200
[Asserts]
jsonpath "$.password_cuenta" == null


# ---------- 11. Estado invalido (400) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": {{servicio_id}},
  "monto": "100.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "x@test.com",
  "password_cuenta": "clave",
  "estado": "Inventada"
}
HTTP 400
[Asserts]
body contains "no es válido"


# ---------- 11b. Una suscripción nueva no puede crearse pausada (400) y no existe el estado Cancelada ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_servicio": {{servicio_id}},
  "monto": "100.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "estado": "Pausada"
}
HTTP 400
[Asserts]
body contains "siempre en estado Activa"

POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_servicio": {{servicio_id}},
  "monto": "100.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "estado": "Cancelada"
}
HTTP 400
[Asserts]
body contains "no es válido"


# ---------- 12. JSON invalido (400) ----------
POST {{host}}/suscripciones
Content-Type: application/json
```
{ esto no es json}
```
HTTP 400


# ---------- 13. Obtener por ID (200) ----------
GET {{host}}/suscripciones/{{sub_id}}
HTTP 200
[Asserts]
jsonpath "$.id_suscripcion" == {{sub_id}}
jsonpath "$.estado" == "Activa"
jsonpath "$.password_cuenta" == null


# ---------- 14. Obtener inexistente (404) ----------
GET {{host}}/suscripciones/999999
HTTP 404
[Asserts]
body contains "no encontrada"


# ---------- 15. ID no numerico (400) ----------
GET {{host}}/suscripciones/abc
HTTP 400


# ---------- 16. Actualizar monto y estado (200) ----------
PUT {{host}}/suscripciones/{{sub_id}}
Content-Type: application/json
{
  "monto": "1999.50",
  "fecha_vencimiento": "2027-06-30T00:00:00Z",
  "estado": "Pausada",
  "usuario_cuenta": "nuevacuenta@test.com",
  "password_cuenta": "nuevaclave"
}
HTTP 200
[Asserts]
jsonpath "$.id_suscripcion" == {{sub_id}}
jsonpath "$.monto" == "1999.50"
jsonpath "$.estado" == "Pausada"
jsonpath "$.usuario_cuenta" == "nuevacuenta@test.com"
jsonpath "$.fecha_vencimiento" startsWith "2027-06-30"
jsonpath "$.fecha_inicio" startsWith "2026-01-01"
jsonpath "$.password_cuenta" == null


# ---------- 16b. Actualizar sin mandar contraseña conserva la guardada (200) ----------
PUT {{host}}/suscripciones/{{sub_id}}
Content-Type: application/json
{
  "monto": "1999.50",
  "fecha_vencimiento": "2027-06-30T00:00:00Z",
  "estado": "Pausada",
  "usuario_cuenta": "nuevacuenta@test.com"
}
HTTP 200
[Asserts]
jsonpath "$.estado" == "Pausada"
jsonpath "$.password_cuenta" == null


# ---------- 17. Verificar que persistio ----------
GET {{host}}/suscripciones/{{sub_id}}
HTTP 200
[Asserts]
jsonpath "$.estado" == "Pausada"
jsonpath "$.monto" == "1999.50"


# ---------- 17b. Ver la contraseña de la cuenta confirmando la del usuario ----------
# con la contraseña del usuario equivocada no se muestra (403)
POST {{host}}/suscripciones/{{sub_id}}/password
Content-Type: application/json
{
  "password": "incorrecta"
}
HTTP 403
[Asserts]
body contains "no es correcta"

# con la contraseña correcta se devuelve la guardada (200)
POST {{host}}/suscripciones/{{sub_id}}/password
Content-Type: application/json
{
  "password": "secreto123"
}
HTTP 200
[Asserts]
jsonpath "$.password_cuenta" == "nuevaclave"

POST {{host}}/suscripciones/999999/password
Content-Type: application/json
{
  "password": "secreto123"
}
HTTP 404

POST {{host}}/suscripciones/abc/password
Content-Type: application/json
{
  "password": "secreto123"
}
HTTP 400

GET {{host}}/suscripciones/{{sub_id}}/password
HTTP 405

# se pueden cambiar el usuario y la contraseña de la cuenta
PUT {{host}}/suscripciones/{{sub_id}}
Content-Type: application/json
{
  "monto": "1999.50",
  "fecha_vencimiento": "2027-06-30T00:00:00Z",
  "estado": "Pausada",
  "usuario_cuenta": "cambiada@test.com",
  "password_cuenta": "otraclave"
}
HTTP 200
[Asserts]
jsonpath "$.usuario_cuenta" == "cambiada@test.com"
jsonpath "$.password_cuenta" == null

POST {{host}}/suscripciones/{{sub_id}}/password
Content-Type: application/json
{
  "password": "secreto123"
}
HTTP 200
[Asserts]
jsonpath "$.password_cuenta" == "otraclave"


# ---------- 18. Actualizar con estado invalido (400) ----------
PUT {{host}}/suscripciones/{{sub_id}}
Content-Type: application/json
{
  "monto": "100.00",
  "fecha_vencimiento": "2027-06-30T00:00:00Z",
  "estado": "Inventada",
  "usuario_cuenta": "a@test.com",
  "password_cuenta": "clave"
}
HTTP 400
[Asserts]
body contains "no es válido"


# ---------- 19. Actualizar con monto cero (400) ----------
PUT {{host}}/suscripciones/{{sub_id}}
Content-Type: application/json
{
  "monto": "0",
  "fecha_vencimiento": "2027-06-30T00:00:00Z",
  "estado": "Pausada",
  "usuario_cuenta": "a@test.com",
  "password_cuenta": "clave"
}
HTTP 400


# ---------- 20. Vencimiento anterior al inicio (400) ----------
PUT {{host}}/suscripciones/{{sub_id}}
Content-Type: application/json
{
  "monto": "100.00",
  "fecha_vencimiento": "2025-01-01T00:00:00Z",
  "estado": "Pausada",
  "usuario_cuenta": "a@test.com",
  "password_cuenta": "clave"
}
HTTP 400
[Asserts]
body contains "anterior"


# ---------- 21. Actualizar inexistente (404) ----------
PUT {{host}}/suscripciones/999999
Content-Type: application/json
{
  "monto": "100.00",
  "fecha_vencimiento": "2027-06-30T00:00:00Z",
  "estado": "Pausada",
  "usuario_cuenta": "a@test.com",
  "password_cuenta": "clave"
}
HTTP 404


# ---------- 22. Cambiar a una cuenta que ya usa otra suscripción (409) ----------
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": {{servicio_id}},
  "monto": "800.00",
  "fecha_inicio": "2026-02-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "segunda@test.com",
  "password_cuenta": "clave2",
  "estado": "Activa"
}
HTTP 201
[Captures]
sub2_id: jsonpath "$.id_suscripcion"

PUT {{host}}/suscripciones/{{sub2_id}}
Content-Type: application/json
{
  "monto": "800.00",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "estado": "Activa",
  "usuario_cuenta": "otra@test.com"
}
HTTP 409
[Asserts]
body contains "con la cuenta"


# ---------- 23. Pausar la segunda y reactivar la primera (200) ----------
PUT {{host}}/suscripciones/{{sub2_id}}
Content-Type: application/json
{
  "monto": "800.00",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "estado": "Pausada",
  "usuario_cuenta": "segunda@test.com",
  "password_cuenta": "clave2"
}
HTTP 200
[Asserts]
jsonpath "$.estado" == "Pausada"

# ya no se puede cancelar: para dejarla se elimina (400)
PUT {{host}}/suscripciones/{{sub2_id}}
Content-Type: application/json
{
  "monto": "800.00",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "estado": "Cancelada",
  "usuario_cuenta": "segunda@test.com"
}
HTTP 400
[Asserts]
body contains "no es válido"

PUT {{host}}/suscripciones/{{sub_id}}
Content-Type: application/json
{
  "monto": "1999.50",
  "fecha_vencimiento": "2027-06-30T00:00:00Z",
  "estado": "Activa",
  "usuario_cuenta": "nuevacuenta@test.com",
  "password_cuenta": "nuevaclave"
}
HTTP 200
[Asserts]
jsonpath "$.estado" == "Activa"


# ---------- 24. Listar por usuario (200) ----------
GET {{host}}/suscripciones/usuario/{{usuario_id}}
HTTP 200
[Asserts]
jsonpath "$" count == 4
jsonpath "$[0].servicio_nombre" exists
jsonpath "$[0].servicio_categoria" == "Streaming"
jsonpath "$[0].password_cuenta" not exists


# ---------- 24c. Renovación automática y períodos de solo lectura ----------
# una suscripción anual que venció en 2025: al listar se renueva sola, por el
# mismo plazo (un año), hasta llegar a un período vigente
GET {{host}}/servicios
HTTP 200
[Captures]
spotify_id: jsonpath "$[?(@.nombre=='Spotify')].id_servicio" nth 0

POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_servicio": {{spotify_id}},
  "monto": "2900.00",
  "fecha_inicio": "2024-03-10T00:00:00Z",
  "fecha_vencimiento": "2025-03-10T00:00:00Z",
  "usuario_cuenta": "renueva@test.com",
  "password_cuenta": "clave"
}
HTTP 201
[Captures]
sub_vieja_id: jsonpath "$.id_suscripcion"

GET {{host}}/suscripciones/usuario/{{usuario_id}}
HTTP 200
[Asserts]
# la cadena de períodos mantiene el día y el monto
jsonpath "$[?(@.servicio_nombre=='Spotify')].fecha_vencimiento" contains "2025-03-10T00:00:00Z"
jsonpath "$[?(@.servicio_nombre=='Spotify')].fecha_vencimiento" contains "2026-03-10T00:00:00Z"
jsonpath "$[?(@.servicio_nombre=='Spotify')].fecha_vencimiento" contains "2027-03-10T00:00:00Z"
jsonpath "$[?(@.servicio_nombre=='Spotify')].fecha_inicio" contains "2026-03-10T00:00:00Z"
jsonpath "$[?(@.servicio_nombre=='Spotify')].monto" contains "2900.00"
jsonpath "$[?(@.estado=='Vencida')].servicio_nombre" contains "Spotify"
jsonpath "$[?(@.estado=='Activa')].servicio_nombre" contains "Spotify"

# el período original quedó como Vencida
GET {{host}}/suscripciones/{{sub_vieja_id}}
HTTP 200
[Asserts]
jsonpath "$.estado" == "Vencida"

# y es de solo lectura: no se puede modificar ni borrar (403)
PUT {{host}}/suscripciones/{{sub_vieja_id}}
Content-Type: application/json
{
  "monto": "1.00",
  "fecha_vencimiento": "2025-03-10T00:00:00Z",
  "estado": "Activa",
  "usuario_cuenta": "renueva@test.com"
}
HTTP 403
[Asserts]
body contains "no se pueden modificar"

DELETE {{host}}/suscripciones/{{sub_vieja_id}}
HTTP 403

# no se puede pasar a Vencida a mano
PUT {{host}}/suscripciones/{{sub_id}}
Content-Type: application/json
{
  "monto": "1999.50",
  "fecha_vencimiento": "2027-06-30T00:00:00Z",
  "estado": "Vencida",
  "usuario_cuenta": "nuevacuenta@test.com"
}
HTTP 400


# ---------- 24b. Otro consumidor no ve ni toca suscripciones ajenas ----------
POST {{host}}/usuarios
Content-Type: application/json
{
  "nombre": "Otro Consumidor",
  "email": "otro_{{newUuid}}@test.com",
  "password": "secreto123"
}
HTTP 201
[Captures]
otro_id: jsonpath "$.id_usuario"
otro_email: jsonpath "$.email"

# --- login como otro consumidor ---
POST {{host}}/sesion
Content-Type: application/json
{
  "email": "{{otro_email}}",
  "password": "secreto123"
}
HTTP 200

GET {{host}}/suscripciones/{{sub_id}}
HTTP 404

PUT {{host}}/suscripciones/{{sub_id}}
Content-Type: application/json
{
  "monto": "1.00",
  "fecha_vencimiento": "2027-06-30T00:00:00Z",
  "estado": "Pausada",
  "usuario_cuenta": "x@test.com",
  "password_cuenta": "x"
}
HTTP 404

DELETE {{host}}/suscripciones/{{sub_id}}
HTTP 404

GET {{host}}/suscripciones/usuario/{{usuario_id}}
HTTP 403

# tampoco puede ver la contraseña de la cuenta, aunque confirme la suya (404)
POST {{host}}/suscripciones/{{sub_id}}/password
Content-Type: application/json
{
  "password": "secreto123"
}
HTTP 404

# sin sesión (401)
DELETE {{host}}/sesion
HTTP 204

GET {{host}}/suscripciones/{{sub_id}}
HTTP 401

GET {{host}}/suscripciones/usuario/{{usuario_id}}
HTTP 401

POST {{host}}/suscripciones/{{sub_id}}/password
Content-Type: application/json
{
  "password": "secreto123"
}
HTTP 401

# --- vuelvo al dueño de las suscripciones ---
POST {{host}}/sesion
Content-Type: application/json
{
  "email": "{{usuario_email}}",
  "password": "secreto123"
}
HTTP 200


# ---------- 25. Metodos no permitidos (405) ----------
GET {{host}}/suscripciones
HTTP 405

PATCH {{host}}/suscripciones/{{sub_id}}
HTTP 405

POST {{host}}/suscripciones/usuario/{{usuario_id}}
HTTP 405


# ---------- 26. Eliminar (204) y verificar (404) ----------
DELETE {{host}}/suscripciones/{{sub2_id}}
HTTP 204

GET {{host}}/suscripciones/{{sub2_id}}
HTTP 404

DELETE {{host}}/suscripciones/{{sub2_id}}
HTTP 404


# ---------- Limpieza ----------
# Borrar el usuario elimina la suscripcion restante (CASCADE)
DELETE {{host}}/usuarios/{{usuario_id}}
HTTP 204

# --- el otro consumidor borra su cuenta ---
POST {{host}}/sesion
Content-Type: application/json
{
  "email": "{{otro_email}}",
  "password": "secreto123"
}
HTTP 200

DELETE {{host}}/usuarios/{{otro_id}}
HTTP 204
