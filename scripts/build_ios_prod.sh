#!/usr/bin/env bash
set -euo pipefail
API_URL="https://emma.drleonardoherrera.com"
APP_ID="com.ema.educacion"

# Opcional: APP_ENV=prod para evitar forzar local en debug (no aplica en release pero mantenido por consistencia)

echo "[build_ios_prod] Cleaning..."
flutter clean >/dev/null
flutter pub get >/dev/null

# --no-codesign para generar el .app sin firmar. Para IPA firmada luego usar xcodebuild / fastlane.
echo "[build_ios_prod] Building iOS release (no-codesign) pointing to $API_URL ..."
flutter build ios --release --no-codesign --dart-define=API_BASE_URL=$API_URL

APP_PATH="build/ios/iphoneos/Runner.app"
if [ ! -d "$APP_PATH" ]; then
  echo "[build_ios_prod] ERROR: Runner.app no encontrado" >&2; exit 1;
fi

# Mostrar tamaño aproximado
APP_SIZE=$(du -sh "$APP_PATH" | cut -f1)
echo "[build_ios_prod] App generada: $APP_PATH (tamaño: $APP_SIZE)"

# Extraer Info.plist para validar bundle id y endpoint embed (solo endpoint se verifica indirectamente)
if command -v plutil >/dev/null; then
  BUNDLE_ID=$(plutil -extract CFBundleIdentifier raw -o - "$APP_PATH/Info.plist" || echo "(desconocido)")
  echo "[build_ios_prod] Bundle Identifier: $BUNDLE_ID"
fi

cat <<EOF
[build_ios_prod] Siguientes pasos para generar IPA firmada:
  1. Abrir ios/Runner.xcworkspace en Xcode.
  2. Seleccionar destino "Any iOS Device (arm64)".
  3. Configurar equipo de firma y certificados.
  4. Product > Archive.
  5. Exportar IPA para App Store / Ad Hoc / Enterprise según corresponda.

Alternativa CLI rápida (requiere configuración de firma):
  xcodebuild -workspace ios/Runner.xcworkspace -scheme Runner -configuration Release -archivePath build/ios/archive/Runner.xcarchive archive
  xcodebuild -exportArchive -archivePath build/ios/archive/Runner.xcarchive -exportOptionsPlist ExportOptions.plist -exportPath build/ios/export
EOF

echo "[build_ios_prod] OK"
