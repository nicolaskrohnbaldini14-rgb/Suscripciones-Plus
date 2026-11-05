# 1. Servicio del catálogo precargado
GET {{host}}/servicios
HTTP 200
[Captures]
servicio_id: jsonpath "$[?(@.nombre=='HBO Max')].id_servicio" nth 0

# 2. Consumidor con sesión iniciada
POST {{host}}/usuarios
Content-Type: application/json
{
  "nombre": "Usuario Pagador",
  "email": "pago_{{newUuid}}@test.com",
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

# 3. Crear Suscripción previa (Vencimiento: 2026-12-31)
POST {{host}}/suscripciones
Content-Type: application/json
{
  "id_usuario": {{usuario_id}},
  "id_servicio": {{servicio_id}},
  "monto": "2500.00",
  "fecha_inicio": "2026-01-01T00:00:00Z",
  "fecha_vencimiento": "2026-12-31T00:00:00Z",
  "usuario_cuenta": "pagos@test.com",
  "password_cuenta": "clave123",
  "estado": "Activa"
}
HTTP 201
[Captures]
suscripcion_id: jsonpath "$.id_suscripcion"


### ============================================================================
### CREACIÓN DE PAGO (POST /pagos)
### ============================================================================

### ---------- 1. Crear pago exitoso (201) ----------
POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": {{suscripcion_id}},
  "monto": "2500.00",
  "fecha_pago": "2026-06-15T00:00:00Z"
}
HTTP 201
[Captures]
pago_id: jsonpath "$.id_pago"
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$.id_pago" > 0
jsonpath "$.id_suscripcion" == {{suscripcion_id}}
jsonpath "$.monto" == "2500.00"
jsonpath "$.fecha_pago" startsWith "2026-06-15"
jsonpath "$.created_at" exists

### ---------- 2. Suscripción inexistente (404) ----------
POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": 999999,
  "monto": "1000.00",
  "fecha_pago": "2026-06-15T00:00:00Z"
}
HTTP 404
[Asserts]
body contains "no existe"

### ---------- 3. ID de suscripción inválido (400) ----------
POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": 0,
  "monto": "1000.00",
  "fecha_pago": "2026-06-15T00:00:00Z"
}
HTTP 400
[Asserts]
body contains "válido"

### ---------- 4. Monto cero o negativo (400) ----------
POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": {{suscripcion_id}},
  "monto": "0",
  "fecha_pago": "2026-06-15T00:00:00Z"
}
HTTP 400
[Asserts]
body contains "mayor a cero"

# un monto que no entra en la base (DECIMAL(10,2)) es un error del usuario (400)
POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": {{suscripcion_id}},
  "monto": "99999999999",
  "fecha_pago": "2026-06-15T00:00:00Z"
}
HTTP 400
[Asserts]
body contains "hasta $ 99.999.999,99"

### ---------- 5. Monto no numérico (400) ----------
POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": {{suscripcion_id}},
  "monto": "abc",
  "fecha_pago": "2026-06-15T00:00:00Z"
}
HTTP 400
[Asserts]
body contains "número válido"

### ---------- 6. Fecha de pago posterior al vencimiento de la suscripción (400) ----------
# La suscripción vence el 2026-12-31; enviamos fecha de pago en 2027
POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": {{suscripcion_id}},
  "monto": "2500.00",
  "fecha_pago": "2027-01-15T00:00:00Z"
}
HTTP 400
[Asserts]
body contains "posterior"

### ---------- 7. Sin fecha de pago (400) ----------
POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": {{suscripcion_id}},
  "monto": "2500.00"
}
HTTP 400
[Asserts]
body contains "obligatoria"

### ---------- 8. JSON inválido (400) ----------
POST {{host}}/pagos
Content-Type: application/json
```
{ esto no es json }
```
HTTP 400


### ============================================================================
### LECTURA DE PAGOS (GET)
### ============================================================================

### ---------- 9. Obtener pago por ID (200) ----------
GET {{host}}/pagos/{{pago_id}}
HTTP 200
[Asserts]
jsonpath "$.id_pago" == {{pago_id}}
jsonpath "$.id_suscripcion" == {{suscripcion_id}}
jsonpath "$.monto" == "2500.00"

### ---------- 10. Obtener pago inexistente (404) ----------
GET {{host}}/pagos/999999
HTTP 404
[Asserts]
body contains "no encontrado"

### ---------- 11. ID de pago no numérico (400) ----------
GET {{host}}/pagos/abc
HTTP 400

### ---------- 12. Listar historial de pagos por suscripción (200) ----------
GET {{host}}/pagos/suscripcion/{{suscripcion_id}}
HTTP 200
[Asserts]
jsonpath "$" count >= 1
jsonpath "$[0].id_suscripcion" == {{suscripcion_id}}

### ---------- 13. Listar historial de suscripción inexistente (404) ----------
GET {{host}}/pagos/suscripcion/999999
HTTP 404
[Asserts]
body contains "no existe"


### ============================================================================
### PAGOS AJENOS Y SIN SESIÓN
### ============================================================================

POST {{host}}/usuarios
Content-Type: application/json
{
  "nombre": "Otro Pagador",
  "email": "otropago_{{newUuid}}@test.com",
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

GET {{host}}/pagos/{{pago_id}}
HTTP 404

GET {{host}}/pagos/suscripcion/{{suscripcion_id}}
HTTP 404

DELETE {{host}}/pagos/{{pago_id}}
HTTP 404

POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": {{suscripcion_id}},
  "monto": "100.00",
  "fecha_pago": "2026-06-15T00:00:00Z"
}
HTTP 404

DELETE {{host}}/sesion
HTTP 204

GET {{host}}/pagos/{{pago_id}}
HTTP 401

POST {{host}}/pagos
Content-Type: application/json
{
  "id_suscripcion": {{suscripcion_id}},
  "monto": "100.00",
  "fecha_pago": "2026-06-15T00:00:00Z"
}
HTTP 401

# --- vuelvo al dueño del pago ---
POST {{host}}/sesion
Content-Type: application/json
{
  "email": "{{usuario_email}}",
  "password": "secreto123"
}
HTTP 200


### ============================================================================
### REGLA DE INMUTABILIDAD DE REGISTROS DE PAGO (PUT / PATCH)
### ============================================================================

### ---------- 14. Intento de modificación / PUT no permitido (405) ----------
PUT {{host}}/pagos/{{pago_id}}
Content-Type: application/json
{
  "monto": "3000.00"
}
HTTP 405
[Asserts]
body contains "inmutables"

### ---------- 15. Intento de modificación / PATCH no permitido (405) ----------
PATCH {{host}}/pagos/{{pago_id}}
HTTP 405


### ============================================================================
### ELIMINACIÓN Y BORRADO EN CASCADA (DELETE)
### ============================================================================

### ---------- 16. Eliminar pago (204) y verificar su ausencia (404) ----------
DELETE {{host}}/pagos/{{pago_id}}
HTTP 204

GET {{host}}/pagos/{{pago_id}}
HTTP 404
[Asserts]
body contains "no encontrado"

# Re-intentar eliminar ya borrado
DELETE {{host}}/pagos/{{pago_id}}
HTTP 404


### ============================================================================
### LIMPIEZA
### ============================================================================

# Borrar usuario elimina en cascada sus suscripciones y pagos asociados (ON DELETE CASCADE)
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
