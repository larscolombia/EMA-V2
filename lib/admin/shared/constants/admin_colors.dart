import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';

/// Colores del panel administrativo
/// Reutiliza la paleta de la app móvil para mantener consistencia visual
class AdminColors {
  // Colores primarios (reutilizando AppStyles)
  static const Color primary = AppStyles.primaryColor; // RGB(0, 51, 163)
  static const Color primaryDark = AppStyles.primary900; // RGB(58, 12, 140)
  static const Color primaryLight = AppStyles.primary100; // RGB(112, 247, 240)

  // Colores secundarios (reutilizando AppStyles)
  static const Color secondary = AppStyles.secondaryColor; // RGB(193, 113, 238)
  static const Color tertiary = AppStyles.tertiaryColor; // RGB(147, 101, 245)

  // Colores de estado
  static const Color success = Color(0xFF4CAF50);
  static const Color warning = Color(0xFFFFC107);
  static const Color error = AppStyles.redColor;
  static const Color info = AppStyles.primary500;

  // Colores de fondo
  static const Color background = AppStyles.grey240;
  static const Color surface = AppStyles.whiteColor;
  static const Color surfaceDark = AppStyles.grey220;

  // Colores de texto
  static const Color textPrimary = AppStyles.primary900;
  static const Color textSecondary = AppStyles.grey150;
  static const Color textHint = AppStyles.greyColor;
  static const Color textOnPrimary = AppStyles.whiteColor;

  // Colores de bordes y divisores
  static const Color divider = AppStyles.grey200;
  static const Color border = AppStyles.grey220;

  // Sidebar (usando paleta de la app)
  static const Color sidebarBackground = AppStyles.primary900; // Morado oscuro
  static const Color sidebarText = AppStyles.grey200;
  static const Color sidebarTextActive = AppStyles.whiteColor;
  static const Color sidebarItemHover = AppStyles.tertiaryColor;
  static const Color sidebarItemActive = AppStyles.secondaryColor;

  // Charts y gráficos (usando paleta de la app)
  static const Color chart1 = AppStyles.primaryColor;
  static const Color chart2 = AppStyles.secondaryColor;
  static const Color chart3 = AppStyles.tertiaryColor;
  static const Color chart4 = AppStyles.primary500;
  static const Color chart5 = AppStyles.primary100;

  static List<Color> get chartColors => [
    chart1,
    chart2,
    chart3,
    chart4,
    chart5,
  ];
}
