# Páginas públicas y control de acceso sin sesión.

# ---------- 1. Home: portada, links de acceso y catálogo en el panel ----------
GET {{host}}/
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
xpath "normalize-space(//section[@class='portada']/h1)" == "Suscripciones+"
xpath "count(//a[@href='/login'])" >= 1
xpath "count(//a[@href='/registro'])" >= 1
# panel de servicios con las 10 categorías del catálogo, cada una con su logo
xpath "count(//div[contains(@class,'panel-servicios')]//section[@class='panel-categoria'])" == 10
xpath "count(//section[@class='panel-categoria']/h3/span[@class='icono-categoria']/*[local-name()='svg'])" == 10
# servicios del hogar y seguros
xpath "count(//section[@class='panel-categoria'][h3[contains(normalize-space(.),'Hogar')]]//*[normalize-space(.)='Luz'])" >= 1
xpath "count(//section[@class='panel-categoria'][h3[contains(normalize-space(.),'Seguros')]]//*[normalize-space(.)='Seguro de auto'])" >= 1
xpath "count(//section[@class='panel-categoria']//*[normalize-space(.)='Netflix'])" >= 1
xpath "count(//section[@class='panel-categoria']//*[normalize-space(.)='Pádel'])" >= 1
# sin sesión los servicios no son links para suscribirse
xpath "count(//a[contains(@href,'/cuenta/suscripciones/nueva?servicio=')])" == 0
# no hay JavaScript
xpath "count(//script)" == 0


# ---------- 2. Páginas de login y registro ----------
GET {{host}}/login
HTTP 200
[Asserts]
xpath "count(//form[@action='/login']//input[@name='email'])" == 1
xpath "count(//form[@action='/login']//input[@name='password'])" == 1

GET {{host}}/registro
HTTP 200
[Asserts]
xpath "count(//form[@action='/registro']//input[@name='nombre'])" == 1


# ---------- 3. Sin sesión, las páginas privadas redirigen al login ----------
GET {{host}}/cuenta
HTTP 303
[Asserts]
header "Location" == "/login"

GET {{host}}/cuenta/suscripciones
HTTP 303
[Asserts]
header "Location" == "/login"

GET {{host}}/cuenta/suscripciones/nueva
HTTP 303
[Asserts]
header "Location" == "/login"

POST {{host}}/cuenta/suscripciones
[FormParams]
id_servicio: 1
HTTP 303
[Asserts]
header "Location" == "/login"

POST {{host}}/cuenta/suscripciones/1/eliminar
HTTP 303
[Asserts]
header "Location" == "/login"


# ---------- 4. Ya no existen el panel ni el registro de empresas ----------
GET {{host}}/empresa
HTTP 404

GET {{host}}/empresas/registro
HTTP 404


# ---------- 5. Estilos y rutas inexistentes ----------
GET {{host}}/static/styles.css
HTTP 200
[Asserts]
header "Content-Type" contains "text/css"
header "Cache-Control" == "no-cache"

GET {{host}}/no-existe
HTTP 404

GET {{host}}/logout
HTTP 405
