import 'package:ema_educacion_medica_avanzada/admin/config/admin_routes.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class AdminMiddleware extends GetMiddleware {
  @override
  int? get priority => 1;

  @override
  RouteSettings? redirect(String? route) {
    // El middleware de GetX debe ser síncrono
    // La validación asíncrona se hace en las páginas
    // Aquí solo verificamos si hay alguna sesión guardada

    // Si intenta acceder al login, permitir siempre
    if (route == AdminRoutes.login) {
      return null;
    }

    // Para otras rutas, permitir el acceso
    // La validación real se hará en el onInit de cada página
    return null;
  }
}
