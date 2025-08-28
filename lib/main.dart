// Todo: https://pub.dev/packages/google_fonts#licensing-fonts
// https://pub.dev/documentation/google_fonts/latest/google_fonts/GoogleFonts/config.html
import 'package:ema_educacion_medica_avanzada/app/app_ui.dart';
import 'package:flutter/material.dart';
import 'config/constants/constants.dart';
import 'package:google_fonts/google_fonts.dart';


void main() async {
  GoogleFonts.config.allowRuntimeFetching = false;

  WidgetsFlutterBinding.ensureInitialized();

  // Log de URL base en arranque para diagnóstico.
  // Ignorar en producción si se desea eliminar después.
  // Muestra si se aplicó correctamente --dart-define=API_BASE_URL.
  // También reflejará si FORCE_LOCAL=1 forzó local.
  // Ejemplo esperado: https://emma.drleonardoherrera.com
  // (En debug sin define y con FORCE_LOCAL=1 mostraría http://localhost:8080)
  // Se puede buscar este print en la consola: [BOOT] apiUrl=...
  // Nota: apiUrl se evalúa una sola vez en constants.dart
  // por lo que hot reload no cambiará este valor sin full restart.
  // REMOVE-ME cuando ya no se necesite.
  // ignore: avoid_print
  print('[BOOT] apiUrl=' + apiUrl);

  runApp(const AppUi());
}
