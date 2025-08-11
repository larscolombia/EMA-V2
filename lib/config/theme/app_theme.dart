import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';


class AppTheme {
  static ThemeData getTheme() {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,

      primaryColor: AppStyles.primaryColor,
      primaryColorDark: AppStyles.primary900,
      colorScheme: ColorScheme.fromSeed(seedColor: AppStyles.primaryColor),
      
      textTheme: GoogleFonts.ralewayTextTheme(),
      appBarTheme: AppBarTheme(
        color: Colors.white,
        elevation: 1,
        iconTheme: IconThemeData(
          color: AppStyles.primaryColor,
        ),
      ),

      bottomSheetTheme: BottomSheetThemeData(
        backgroundColor: AppStyles.whiteColor,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(0),
        ),
      ),
    );
  }
}
