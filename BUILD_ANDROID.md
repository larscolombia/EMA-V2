# Build APK Produccion

Para generar un APK conectado al backend de producción:

1. Asegúrate de que `lib/config/constants/constants.dart` tiene `useLocalBackend = false` (ya está por defecto) o usa dart-define.
2. (Opcional) Para forzar otra URL sin tocar código:

```
flutter build apk --release --dart-define=API_BASE_URL=https://emma.drleonardoherrera.com
```

Si quieres apuntar a staging:
```
flutter build apk --release --dart-define=API_BASE_URL=https://staging.tu-dominio.com
```

El APK resultante queda en `build/app/outputs/flutter-apk/app-release.apk`.
