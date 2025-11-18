# Configuración y Pruebas - Recuperación de Contraseña

## Configuración del Backend

### Variables de entorno en .env

Asegúrate de tener configuradas las siguientes variables:

1. **Configuración SMTP** (para envío de correos):
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USER=pruebas.governor@gmail.com
   SMTP_PASS=vneftnzkhfyddkxs
   SMTP_FROM=pruebas.governor@gmail.com

2. **URL del Frontend** (para enlaces de recuperación):
   FRONTEND_URL=http://localhost:5173
   
   En producción cambiar a:
   FRONTEND_URL=https://emma.drleonardoherrera.com

### Tablas de Base de Datos

La tabla password_resets se creará automáticamente al ejecutar el backend.

## Flujo de Recuperación de Contraseña

### 1. Solicitud de Recuperación
Endpoint: POST /password/forgot

### 2. Cambio de Contraseña
Endpoint: POST /password/reset

## Pruebas Locales

1. Iniciar el Backend: cd backend; go run .
2. Iniciar el Frontend Flutter
3. Navegar a pantalla de login
4. Click en ¿Olvidaste tu contraseña?
5. Ingresar email registrado
6. Revisar el correo
7. Click en el enlace de recuperación
8. Ingresar nueva contraseña
9. Verificar redirección a login

## Seguridad Implementada

- Tokens únicos de 32 caracteres
- Expiración de 1 hora
- Tokens de un solo uso
- Prevención de enumeración de usuarios
- Validación de longitud mínima de contraseña
- Emails de confirmación
