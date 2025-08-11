// Todo: https://pub.dev/packages/google_fonts#licensing-fonts
// https://pub.dev/documentation/google_fonts/latest/google_fonts/GoogleFonts/config.html
import 'package:ema_educacion_medica_avanzada/app/app_ui.dart';
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';


void main() async {
  GoogleFonts.config.allowRuntimeFetching = false;

  WidgetsFlutterBinding.ensureInitialized();

  runApp(const AppUi());
}
