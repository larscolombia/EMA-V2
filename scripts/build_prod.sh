#!/usr/bin/env bash
set -euo pipefail
API_URL="https://emma.drleonardoherrera.com"
APP_ID="com.ema.educacion"

echo "[build_prod] Cleaning..."
flutter clean >/dev/null
flutter pub get >/dev/null

echo "[build_prod] Building release APK pointing to $API_URL ..."
flutter build apk --release --dart-define=API_BASE_URL=$API_URL

APK=build/app/outputs/flutter-apk/app-release.apk

if [ ! -f "$APK" ]; then
  echo "[build_prod] ERROR: APK no encontrado" >&2; exit 1;
fi

# Verificación básica de integridad
file "$APK" | grep -qi 'Zip archive' || { echo "[build_prod] ERROR: APK corrupto?" >&2; exit 1; }

# Mostrar applicationId esperado dentro del AndroidManifest (aapt dump badging si disponible)
if command -v aapt >/dev/null; then
  echo "[build_prod] Verificando applicationId..."
  aapt dump badging "$APK" | grep package:
fi

# Hash para comparar en transferencias
sha1sum "$APK" | awk '{print "[build_prod] SHA1=" $1}'

echo "[build_prod] OK -> $APK"
