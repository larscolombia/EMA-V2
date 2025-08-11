import 'dart:io' show Platform;
import 'package:flutter/foundation.dart';

/// URL base para el backend en Go (detecta emulador de Android)
final String apiUrl = () {
	if (kIsWeb) return 'http://localhost:8080';
	try {
		if (Platform.isAndroid) return 'http://10.0.2.2:8080';
	} catch (_) {
		// Platform may not be available in some contexts; fallback
	}
	return 'http://localhost:8080';
}();

/// Habilita toda la funcionalidad para pruebas
/// `lib/config/constants/constants.dart`.
const bool useAllFeatures = false;
