# Usuarios de prueba del .env (.env.example): se crean al iniciar la app con
# suscripciones de ejemplo, para poder probarla sin registrarse.

# ---------- Martina entra con los datos del .env ----------
POST {{host}}/sesion
Content-Type: application/json
{
  "email": "martina@ejemplo.com",
  "password": "martina123"
}
HTTP 200
[Captures]
martina_id: jsonpath "$.id_usuario"
[Asserts]
jsonpath "$.nombre" == "Martina"

# 6 suscripciones de ejemplo + la renovación automática de la del mes anterior
GET {{host}}/suscripciones/usuario/{{martina_id}}
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
  "email": "agustina@ejemplo.com",
  "password": "agustina123"
}
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Agustina"

DELETE {{host}}/sesion
HTTP 204

POST {{host}}/sesion
Content-Type: application/json
{
  "email": "nicolas@ejemplo.com",
  "password": "nicolas123"
}
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Nicolás"

DELETE {{host}}/sesion
HTTP 204
