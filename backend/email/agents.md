# email package: SMTP helpers

Environment variables
- SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS: servidor SMTP para enviar correos.
- SMTP_FROM (opcional): dirección de origen; si se omite se usa SMTP_USER.

How it works
- Funciones `SendWelcome` y `SendPasswordChanged` construyen y envían correos.
- Los errores se retornan para que las llamadas los registren sin detener la ejecución.
- `SendUpgradeSuggestion` envía promociones para cambiar a planes premium.
