# Build APK Produccion

Para generar un APK apuntando a backend de producción:

1. El fallback en código ya es producción: `https://emma.drleonardoherrera.com`.
2. En debug se fuerza local (emulador) si no pasas nada. En release NO se fuerza local.
3. Para asegurar explícitamente la URL (recomendado en CI):

```
flutter build apk --release --dart-define=API_BASE_URL=https://emma.drleonardoherrera.com
```

Staging (ejemplo):
```
flutter build apk --release --dart-define=API_BASE_URL=https://staging.tu-dominio.com
```

Forzar backend local (solo pruebas internas) incluso en release:
```
flutter build apk --release --dart-define=APP_ENV=dev
```

Salida: `build/app/outputs/flutter-apk/app-release.apk`.

## Script rápido

También puedes usar el script automatizado (valida integridad y muestra SHA1):

```
bash scripts/build_prod.sh
```

## Logs en release

`Logger.debug/info` se silencian en release; sólo `warn/error` salen. Usa `Logger.debug` para mensajes de diagnóstico sin ruido en producción.
