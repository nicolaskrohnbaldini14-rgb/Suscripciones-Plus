# Usuarios de prueba del .env (.env.example): se crean al iniciar la app con
# suscripciones de ejemplo, para poder probarla sin registrarse.

# ---------- Martina entra con los datos del .env ----------
POST {{host}}/login
[Options]
location: true
[FormParams]
email: martina@ejemplo.com
password: martina123
HTTP 200
[Asserts]
url == "{{host}}/cuenta"
xpath "normalize-space(//main//h1)" == "Hola, Martina"
# en /cuenta, debajo de las opciones, todas sus suscripciones activas (la
# pausada de Microsoft 365 y el período vencido de OSDE no aparecen)
xpath "normalize-space(//section[contains(@class,'activas')]/h2)" == "Tus suscripciones activas"
xpath "count(//ul[@id='lista-activas']/li)" == 5
xpath "count(//ul[@id='lista-activas']//strong[starts-with(normalize-space(.),'Microsoft 365')])" == 0

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
email: agustina@ejemplo.com
password: agustina123
HTTP 200
[Asserts]
xpath "normalize-space(//main//h1)" == "Hola, Agustina"

POST {{host}}/logout
HTTP 303

POST {{host}}/login
[Options]
location: true
[FormParams]
email: nicolas@ejemplo.com
password: nicolas123
HTTP 200
[Asserts]
xpath "normalize-space(//main//h1)" == "Hola, Nicolás"

POST {{host}}/logout
HTTP 303
