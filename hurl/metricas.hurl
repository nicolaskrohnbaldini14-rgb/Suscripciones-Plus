# Monitoreo (Prometheus): GET /metrics expone las métricas en formato texto.
# Corre al final, así ya hay pedidos medidos por los archivos anteriores.

GET {{host}}/metrics
HTTP 200
[Asserts]
header "Content-Type" contains "text/plain"
# marco RED: pedidos (por método, ruta y código), errores (status) y duración
body contains "# TYPE http_requests_total counter"
body contains "# TYPE http_request_duration_seconds histogram"
# la ruta es el patrón (sin ids), para no crear una serie por cada URL
body contains "path=\"DELETE /cuenta/suscripciones/{id}\""
body contains "path=\"POST /cuenta/suscripciones\""
body not contains "path=\"/cuenta/suscripciones/1\""
# métricas del negocio, ya con valores de los tests anteriores
body matches /suscripciones_creadas_total [1-9]/
body matches /suscripciones_eliminadas_total [1-9]/
body matches /inicios_de_sesion_total\{resultado="ok"\} [1-9]/
body matches /inicios_de_sesion_total\{resultado="error"\} [1-9]/
# métricas de Go que agrega el cliente sin escribir código
body contains "go_goroutines"
