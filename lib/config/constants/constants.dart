import 'dart:io' show Platform;
import 'package:flutter/foundation.dart';

/// Permite sobre-escribir la URL base usando --dart-define=API_BASE_URL=https://... al construir.
const String _envApiBase = String.fromEnvironment('API_BASE_URL', defaultValue: '');
/// Bandera local para desarrollo rápido (si no se pasa dart-define y se pone en true)
const bool useLocalBackend = false; // dejar en false para builds de cliente

String _computeLocal() {
  if (kIsWeb) return 'http://localhost:8080';
  try {
    if (Platform.isAndroid) return 'http://10.0.2.2:8080';
  } catch (_) {}
  return 'http://localhost:8080';
}

final String apiUrl = () {
  // 1. dart-define tiene prioridad
  if (_envApiBase.trim().isNotEmpty) return _envApiBase.trim();
  // 2. bandera local
  if (useLocalBackend) return _computeLocal();
  // 3. fallback producción
  return 'https://emma.drleonardoherrera.com';
}();

/// Habilita toda la funcionalidad para pruebas
/// `lib/config/constants/constants.dart`.
const bool useAllFeatures = false;
