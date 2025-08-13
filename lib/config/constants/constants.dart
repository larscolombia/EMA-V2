import 'dart:io' show Platform;
import 'package:flutter/foundation.dart';

/// URL base para el backend en producción
/// Cambia a true `useLocalBackend` si necesitas apuntar a localhost durante desarrollo.
const bool useLocalBackend = false;

final String apiUrl = () {
  if (useLocalBackend) {
    if (kIsWeb) return 'http://localhost:8080';
    try {
      if (Platform.isAndroid) return 'http://10.0.2.2:8080';
    } catch (_) {}
    return 'http://localhost:8080';
  }
  // Producción
  return 'https://emma.drleonardoherrera.com';
}();

/// Habilita toda la funcionalidad para pruebas
/// `lib/config/constants/constants.dart`.
const bool useAllFeatures = false;
