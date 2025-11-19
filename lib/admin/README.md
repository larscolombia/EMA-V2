# Panel Administrativo Web - EMA

##  Estructura Creada

Se ha implementado la estructura base del panel administrativo web integrado con Flutter Web.

###  Estructura de Carpetas

```
lib/admin/
 config/
    admin_routes.dart          # Rutas del panel admin
    admin_bindings.dart        # Dependencias GetX
 core/
    models/
       admin_user.dart        # Modelo de usuario admin
    services/
       admin_auth_service.dart  # Servicio de autenticación admin
    middleware/
        admin_middleware.dart  # Middleware de protección de rutas
 features/
    auth/
       pages/
          admin_login_page.dart  # Página de login admin
       controllers/
           admin_auth_controller.dart  # Controlador de autenticación
    dashboard/
       pages/
           dashboard_page.dart  # Dashboard principal
    plans/
       pages/
           plans_page.dart  # Gestión de planes
    statistics/
       pages/
           statistics_page.dart  # Estadísticas y métricas
    books/
       pages/
           books_page.dart  # Gestión de libros
    users/
        pages/
            users_page.dart  # Gestión de usuarios
 shared/
     layout/
        admin_layout.dart  # Layout base con sidebar y navbar
     widgets/
        admin_sidebar.dart  # Sidebar responsive
        admin_navbar.dart  # Navbar con perfil
     constants/
         admin_colors.dart  # Paleta de colores profesional
```

###  Sistema de Autenticación

**Características implementadas:**
- Login exclusivo para administradores (`role === 'super_admin'`)
- Validación de rol en backend
- Almacenamiento seguro de tokens con `flutter_secure_storage`
- Middleware de protección de rutas
- Auto-redirect si no es admin
- Logout con limpieza de sesión

**Flujo:**
1. Usuario accede a `/admin/login`
2. Ingresa credenciales
3. Backend valida que sea `super_admin`
4. Si no es admin  Error: "Acceso denegado"
5. Si es admin  Guarda token y redirige a `/admin/dashboard`

###  Diseño y UX

**Características:**
- **Visual consistency**: Reutiliza componentes y estilos de la app móvil
- **BackgroundWidget**: Mismo gradiente púrpura de la app móvil (primary900 → secondaryColor)
- **CustomTextField**: Componentes de entrada idénticos a la app móvil
- **Color palette**: Importa y usa `AppStyles` (primary900, secondaryColor, tertiaryColor, etc.)
- **Responsive**: Adaptado para desktop, tablet y móvil
- **Sidebar colapsable**: Se puede contraer para más espacio
- **Navbar profesional**: Con notificaciones y menú de perfil
- **Componentes reutilizables**: Layout, widgets, estilos

**Breakpoints:**
- Desktop: >= 1024px (sidebar fijo)
- Tablet: 768px - 1023px (drawer)
- Móvil: < 768px (drawer)

**Estilo visual:**
- Logo blanco (logotype_white.png) sobre fondo gradiente
- Card blanco con bordes redondeados (24px)
- Colores primarios: `AppStyles.primary900` (RGB 58,12,140)
- Colores secundarios: `AppStyles.secondaryColor` (RGB 193,113,238)
- Campos de texto con estilo unificado mediante `CustomTextField`

###  Integración

**Rutas integradas en `app_pages.dart`:**
```dart
// Rutas del panel administrativo
...AdminRoutes.routes,
```

**Rutas disponibles:**
- `/admin/login`  Login de administradores
- `/admin/dashboard`  Dashboard principal
- `/admin/users`  Gestión de usuarios
- `/admin/plans`  Gestión de planes
- `/admin/books`  Gestión de libros
- `/admin/statistics`  Estadísticas y métricas

###  Estado Actual

 **Completado:**
- [x] Estructura de carpetas
- [x] Modelo de usuario admin
- [x] Servicio de autenticación
- [x] Login page con diseño profesional
- [x] Layout base (sidebar + navbar)
- [x] Sidebar responsive
- [x] Navbar con menú de perfil
- [x] Middleware de protección
- [x] Sistema de rutas
- [x] Integración con app principal
- [x] Paleta de colores (reutiliza AppStyles)
- [x] Consistencia visual con app móvil
- [x] BackgroundWidget y CustomTextField reutilizados
- [x] Logo EMA en pantalla de login

 **Pendiente (Próximas Implementaciones):**
- [ ] Dashboard con widgets de métricas
- [ ] Módulo de gestión de planes (CRUD)
- [ ] Módulo de estadísticas con gráficas
- [ ] Módulo de gestión de libros
- [ ] Módulo de gestión de usuarios
- [ ] Servicios de API para cada módulo
- [ ] Modelos de datos completos
- [ ] Formularios y validaciones
- [ ] Paginación y búsqueda
- [ ] Exportación de datos

###  Cómo Ejecutar

#### **Paso 1: Asegurar que el Backend está corriendo**

```powershell
# Ir a la carpeta del backend
cd backend

# Asegurar que tienes el archivo .env configurado
# Copia .env.example a .env y configura:
# - DEFAULT_USER_EMAIL=admin@ema.com
# - DEFAULT_USER_PASSWORD=admin123

# Ejecutar el backend
go run main.go
# O si ya está compilado:
.\ema-backend.exe
```

El backend debería mostrar:
```
[MIGRATION] ✅ users table ready
[SEED] Default user created: admin@ema.com
Server running on :8080
```

---

#### **Paso 2: Ejecutar Flutter Web**

```powershell
# Volver a la raíz del proyecto
cd ..

# Opción 1: Desarrollo con API local (backend en localhost:8080)
flutter run -d chrome

# Opción 2: Con API de producción
flutter run -d chrome --dart-define=API_BASE_URL=https://emma.drleonardoherrera.com

# Opción 3: Especificar puerto fijo para facilitar el acceso
flutter run -d chrome --web-port=8080
```

---

#### **Paso 3: Acceder al Panel**

1. El navegador Chrome se abrirá automáticamente
2. Verás en la consola algo como:
   ```
   Flutter run key commands.
   ...
   A Chromium-based browser is required to run Flutter web applications.
   Launching lib\main.dart on Chrome in debug mode...
   ```

3. **Navega a la ruta del admin:**
   - Si usaste `--web-port=8080`: `http://localhost:8080/#/admin/login`
   - Si no especificaste puerto: `http://localhost:[PUERTO_AUTO]/#/admin/login`
   - El puerto automático aparece en la consola (ej: `http://localhost:54321/`)

---

#### **Paso 4: Iniciar Sesión**

**Credenciales del usuario admin (configuradas en backend/.env):**
```
Email: admin@ema.com
Password: admin123
```

**IMPORTANTE:** 
- Solo usuarios con `role = 'super_admin'` pueden acceder
- El backend crea automáticamente este usuario en el primer arranque si no existe
- Si usas otras credenciales, asegúrate de que el rol sea `super_admin` en la BD

---

#### **Compilar para Producción (Opcional)**

```powershell
# Compilar versión optimizada
flutter build web --release --dart-define=API_BASE_URL=https://emma.drleonardoherrera.com

# Los archivos estarán en: build/web/
# Puedes desplegarlos en cualquier hosting (Firebase, Netlify, Vercel, etc.)
```

---

#### **Solución de Problemas**

**Error: "Acceso denegado"**
- Verifica que el usuario tenga `role = 'super_admin'` en la tabla `users`
- Consulta en MySQL: `SELECT id, email, role FROM users WHERE email = 'admin@ema.com';`

**Error: "Token inválido"**
- El token puede haber expirado
- Cierra sesión y vuelve a iniciar

**Backend no responde:**
- Verifica que el backend esté corriendo en `http://localhost:8080`
- Revisa la consola del backend por errores
- Verifica la conexión a MySQL

**No aparece el login:**
- Asegúrate de navegar a `/#/admin/login` (con el hash)
- Verifica en la consola del navegador (F12) si hay errores

###  Próximos Pasos Recomendados

1. **Dashboard (Prioridad Alta)**
   - Cards con métricas clave
   - Gráficos de tendencias
   - Actividad reciente
   - Alertas y notificaciones

2. **Gestión de Planes (Prioridad Alta)**
   - Listado de planes con tabla
   - Crear/Editar plan con formulario
   - Activar/Desactivar planes
   - Ver usuarios asignados
   - Integración con Stripe

3. **Estadísticas (Prioridad Media)**
   - Ventas por período
   - Métricas de uso
   - Planes más vendidos
   - Gráficas interactivas (fl_chart)
   - Exportación a CSV/Excel

4. **Gestión de Libros (Prioridad Media)**
   - Subir archivos (PDF, EPUB)
   - Organizar por categorías
   - Editar metadata
   - Eliminar contenido

5. **Gestión de Usuarios (Prioridad Media)**
   - Listado con búsqueda y filtros
   - Ver detalles de usuario
   - Editar perfiles
   - Cambiar planes
   - Estadísticas por usuario
   - **NOTA**: Cada usuario aparece UNA SOLA VEZ, mostrando su suscripción activa o la más reciente

###  Cambios Recientes

**19 de noviembre de 2025 - Fix: Usuarios duplicados en lista**
- **Problema**: Usuarios con múltiples suscripciones aparecían varias veces en la lista
- **Solución**: Modificado endpoint `/admin/stats/users/list` para retornar un único registro por usuario
- **Criterio**: Prioriza suscripción activa; si tiene varias, muestra la más reciente
- **Archivos modificados**: `backend/stats/admin_stats.go`
- **Query optimizado**: Usa `ROW_NUMBER()` con `PARTITION BY user_id` para deduplicar

###  Tecnologías Utilizadas

- **Flutter Web**: Framework principal
- **GetX**: State management y navegación
- **flutter_secure_storage**: Almacenamiento seguro de tokens
- **http**: Peticiones HTTP
- **fl_chart**: Gráficas (futuro)
- **Backend API**: Go (Gin) con MySQL

###  Notas Importantes

- El panel admin **NO interfiere** con la app móvil
- Usuarios normales **NO pueden** acceder al panel
- Administradores **pueden** acceder solo al panel web
- Todos los componentes son reutilizables y modulares
- Código limpio siguiendo principios SOLID
- Responsive design para todos los dispositivos

---

**Última actualización:** 14 de noviembre de 2025
**Estado:**  Estructura base completada - Listo para implementación de módulos
