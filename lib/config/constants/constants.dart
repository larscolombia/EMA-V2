import 'dart:io' show Platform;
import 'package:flutter/foundation.dart';

/// Permite sobre-escribir la URL base usando --dart-define=API_BASE_URL=https://... al construir.
const String _envApiBase = String.fromEnvironment('API_BASE_URL', defaultValue: '');
/// Fuerza backend local automáticamente en compilaciones debug si no se pasó API_BASE_URL.
/// No afecta a release/profile.
const bool forceLocalInDebug = false; // Desactivado para apuntar a producción por defecto.

String _computeLocal() {
  if (kIsWeb) return 'http://localhost:8080';
  try {
    if (Platform.isAndroid) return 'http://10.0.2.2:8080';
  } catch (_) {}
  return 'http://localhost:8080';
}

// URL base computada (renombrada para evitar choque con getters de ApiConstants)
final String _computedApiUrl = () {
  // 1. dart-define tiene prioridad absoluta
  if (_envApiBase.trim().isNotEmpty) return _envApiBase.trim();
  // 2. En debug podemos forzar local automáticamente
  if (kDebugMode && forceLocalInDebug) return _computeLocal();
  // 3. fallback producción
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
