# Usuarios de prueba del .env (.env.example): se crean al iniciar la app con
# suscripciones de ejemplo, para poder probarla sin registrarse.

# ---------- Ana entra con los datos del .env ----------
POST {{host}}/sesion
Content-Type: application/json
{
  "email": "ana@ejemplo.com",
  "password": "ana12345"
}
HTTP 200
[Captures]
ana_id: jsonpath "$.id_usuario"
[Asserts]
jsonpath "$.nombre" == "Ana Prueba"

# 6 suscripciones de ejemplo + la renovación automática de la del mes anterior
GET {{host}}/suscripciones/usuario/{{ana_id}}
HTTP 200
[Asserts]
jsonpath "$" count == 7
jsonpath "$[?(@.servicio_nombre == 'Netflix')]" count == 2
jsonpath "$[?(@.estado == 'Pausada')]" count == 1
jsonpath "$[?(@.estado == 'Vencida')]" count == 1
# las contraseñas de las cuentas nunca se devuelven
jsonpath "$[*].password_cuenta" not contains "netflix123"

DELETE {{host}}/sesion
HTTP 204


# ---------- Los otros usuarios de prueba también pueden entrar ----------
POST {{host}}/sesion
Content-Type: application/json
{
  "email": "bruno@ejemplo.com",
  "password": "bruno12345"
}
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Bruno Prueba"

DELETE {{host}}/sesion
HTTP 204

POST {{host}}/sesion
Content-Type: application/json
{
  "email": "carla@ejemplo.com",
  "password": "carla12345"
}
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Carla Prueba"

DELETE {{host}}/sesion
HTTP 204
