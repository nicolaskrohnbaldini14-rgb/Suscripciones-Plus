# Catálogo de servicios: viene precargado al iniciar la app y es de solo lectura.

# ---------- 1. Listar el catálogo (200) ----------
GET {{host}}/servicios
HTTP 200
[Captures]
netflix_id: jsonpath "$[?(@.nombre=='Netflix')].id_servicio" nth 0
[Asserts]
header "Content-Type" == "application/json"
jsonpath "$" count >= 47
# algunos servicios precargados por categoría
jsonpath "$[?(@.nombre=='Netflix')].categoria" nth 0 == "Streaming"
jsonpath "$[?(@.nombre=='HBO Max')].categoria" nth 0 == "Streaming"
jsonpath "$[?(@.nombre=='Spotify')].categoria" nth 0 == "Música"
jsonpath "$[?(@.nombre=='iTunes')].categoria" nth 0 == "Música"
jsonpath "$[?(@.nombre=='Gimnasio')].categoria" nth 0 == "Deportes"
jsonpath "$[?(@.nombre=='Pádel')].categoria" nth 0 == "Deportes"
jsonpath "$[?(@.nombre=='OSDE')].categoria" nth 0 == "Salud"
# servicios del hogar y seguros
jsonpath "$[?(@.nombre=='Internet / Wifi')].categoria" nth 0 == "Hogar"
jsonpath "$[?(@.nombre=='Luz')].categoria" nth 0 == "Hogar"
jsonpath "$[?(@.nombre=='Gas')].categoria" nth 0 == "Hogar"
jsonpath "$[?(@.nombre=='Seguro de auto')].categoria" nth 0 == "Seguros"
jsonpath "$[?(@.nombre=='Seguro de hogar')].categoria" nth 0 == "Seguros"
jsonpath "$[?(@.categoria=='Hogar')]" count == 7
jsonpath "$[?(@.categoria=='Seguros')]" count == 6
# la categoría "Gimnasio" pasó a llamarse "Deportes"
jsonpath "$[?(@.categoria=='Gimnasio')]" count == 0
jsonpath "$[?(@.categoria=='Deportes')]" count >= 8


# ---------- 2. Obtener por ID (200) ----------
GET {{host}}/servicios/{{netflix_id}}
HTTP 200
[Asserts]
jsonpath "$.id_servicio" == {{netflix_id}}
jsonpath "$.nombre" == "Netflix"
jsonpath "$.categoria" == "Streaming"


# ---------- 3. Obtener inexistente (404) ----------
GET {{host}}/servicios/999999
HTTP 404
[Asserts]
body contains "no encontrado"


# ---------- 4. ID no numérico (400) ----------
GET {{host}}/servicios/abc
HTTP 400


# ---------- 5. No se pueden crear, modificar ni borrar servicios (405) ----------
POST {{host}}/servicios
Content-Type: application/json
{
  "nombre": "Servicio Nuevo",
  "categoria": "Streaming"
}
HTTP 405
[Asserts]
body contains "precargado"

PUT {{host}}/servicios/{{netflix_id}}
Content-Type: application/json
{
  "nombre": "Netflix Renombrado",
  "categoria": "Streaming"
}
HTTP 405

PATCH {{host}}/servicios/{{netflix_id}}
HTTP 405

DELETE {{host}}/servicios/{{netflix_id}}
HTTP 405

# el servicio sigue en el catálogo
GET {{host}}/servicios/{{netflix_id}}
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Netflix"
