# Health check de la aplicación (salud.go): no necesita sesión.

# ---------- Liveness: el proceso está corriendo ----------
GET {{host}}/salud/vivo
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.estado" == "vivo"

# ---------- Readiness: la base de datos responde ----------
GET {{host}}/salud/lista
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.estado" == "lista"

# ---------- Solo GET ----------
POST {{host}}/salud/vivo
HTTP 405

POST {{host}}/salud/lista
HTTP 405
