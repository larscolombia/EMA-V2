import 'dart:io' show Platform;
import 'package:flutter/foundation.dart';

/// Permite sobre-escribir la URL base usando --dart-define=API_BASE_URL=https://... al construir.
const String _envApiBase = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: '',
);

/// Fuerza backend local automáticamente en compilaciones debug si no se pasó API_BASE_URL.
/// Controlable vía --dart-define=FORCE_LOCAL=1. Por defecto desactivado.
const String _forceLocalFlag = String.fromEnvironment(
  'FORCE_LOCAL',
  defaultValue: '0',
);
const bool forceLocalInDebug = _forceLocalFlag == '1';

String _computeLocal() {
  // Backend corre en puerto 8080
  if (kIsWeb) return 'http://localhost:8080';
  try {
    if (Platform.isAndroid) return 'http://10.0.2.2:8080';
  } catch (_) {}
  return 'http://localhost:8080';
}

// URL base computada (renombrada para evitar choque con getters de ApiConstants)
final String _computedApiUrl = () {
  // Permitir APP_ENV=dev desde --dart-define para forzar local
  const String appEnv = String.fromEnvironment('APP_ENV', defaultValue: '');
  if (appEnv == 'dev') return _computeLocal();
  // 1. dart-define tiene prioridad absoluta
  if (_envApiBase.trim().isNotEmpty) return _envApiBase.trim();
  // 2. En debug podemos forzar local automáticamente con FORCE_LOCAL=1
  if (kDebugMode && forceLocalInDebug) return _computeLocal();
  // 3. Por defecto siempre usar producción
  return 'https://emma.drleonardoherrera.com';
}();

// Mantener alias público previo (si el código usa la variable global apiUrl directamente)
final String apiUrl = _computedApiUrl;

/// Compat layer: algunas partes antiguas del código (o builds intermedios) podrían referirse a
/// `ApiConstants.baseUrl` o `ApiConstants.apiUrl`. Mantener esto evita fallos de build mientras
/// se actualizan referencias. Marcar para eliminación futura.
class ApiConstants {
  // Todos los getters apuntan a la variable global ya evaluada, evitando recursión mutua.
  static String get baseUrl => _computedApiUrl;
  static String get apiBaseUrl => _computedApiUrl; // alias redundante
  static String get apiUrl => _computedApiUrl; // alias legacy
}

/// Habilita toda la funcionalidad para pruebas
/// `lib/config/constants/constants.dart`.
const bool useAllFeatures = false;
