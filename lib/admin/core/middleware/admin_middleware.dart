import 'package:ema_educacion_medica_avanzada/admin/config/admin_routes.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/services/admin_auth_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class AdminMiddleware extends GetMiddleware {
  @override
  int? get priority => 1;

  @override
  RouteSettings? redirect(String? route) {
    final authService = Get.find<AdminAuthService>();

    // Verificar si está autenticado de forma asíncrona
    authService.isAuthenticated().then((isAuth) {
      if (!isAuth && route != AdminRoutes.login) {
        Get.offAllNamed(AdminRoutes.login);
      }
    });

    return null;
  }
}
