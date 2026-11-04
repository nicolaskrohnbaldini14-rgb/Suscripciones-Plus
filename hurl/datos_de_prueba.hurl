# Usuarios de prueba del .env (.env.example): se crean al iniciar la app con
# suscripciones de ejemplo, para poder probarla sin registrarse.

# ---------- Ana entra con los datos del .env ----------
POST {{host}}/login
[Options]
location: true
[FormParams]
email: ana@ejemplo.com
password: ana12345
HTTP 200
[Asserts]
url == "{{host}}/cuenta"
xpath "normalize-space(//main//h1)" == "Hola, Ana Prueba"

# este mes vencen 5 de sus suscripciones de ejemplo (incluida la renovación
# automática de OSDE, que venció el mes anterior)
GET {{host}}/cuenta/suscripciones
HTTP 200
[Asserts]
xpath "count(//ul[@id='lista-suscripciones']/li[contains(@class,'item')])" == 5
xpath "count(//ul[@id='lista-suscripciones']/li[contains(.,'Netflix')])" == 2
xpath "count(//ul[@id='lista-suscripciones']/li[contains(.,'OSDE')])" == 1

POST {{host}}/logout
HTTP 303


# ---------- Los otros usuarios de prueba también pueden entrar ----------
POST {{host}}/login
[Options]
location: true
[FormParams]
email: bruno@ejemplo.com
password: bruno12345
HTTP 200
[Asserts]
xpath "normalize-space(//main//h1)" == "Hola, Bruno Prueba"

POST {{host}}/logout
HTTP 303

POST {{host}}/login
[Options]
location: true
[FormParams]
email: carla@ejemplo.com
password: carla12345
HTTP 200
[Asserts]
xpath "normalize-space(//main//h1)" == "Hola, Carla Prueba"

POST {{host}}/logout
HTTP 303
