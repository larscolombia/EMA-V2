import 'dart:convert'; // Para usar jsonDecode

import 'package:ema_educacion_medica_avanzada/core/auth/laravel_auth_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class ForgotPasswordController extends GetxController {
  final emailController = TextEditingController();
  final LaravelAuthService _authService = Get.find<LaravelAuthService>();

  // Estado de carga
  var isLoading = false.obs;

  // Acción al presionar el botón "Next"
  Future<void> onNextPressed() async {
    final email = emailController.text.trim();

    if (email.isEmpty) {
      Get.snackbar(
        'Error',
        'Por favor ingrese su email',
        backgroundColor: Colors.red.withValues(alpha: 0.8),
        colorText: Colors.white,
      );
      return;
    }

    isLoading.value = true; // Activar carga

    try {
      await _authService.forgotPassword(email);
      Get.snackbar(
        'Éxito',
        'Se ha enviado un enlace de recuperación a su correo electrónico.',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.green.withValues(alpha: 0.8),
        colorText: Colors.white,
      );

      Get.offNamed('/login');
    } catch (e) {
      // Manejo de errores del backend
      final errorMessage = _extractErrorMessage(e);
      Get.snackbar(
        'Error',
        errorMessage,
        backgroundColor: Colors.red.withValues(alpha: 0.8),
        colorText: Colors.white,
      );
    } finally {
      isLoading.value = false; // Desactivar carga
    }
  }

  @override
  void onClose() {
    emailController.dispose();
    super.onClose();
  }

  /// Extrae el mensaje de error del JSON de respuesta.
  String _extractErrorMessage(dynamic error) {
    try {
      // Convierte el error a String (esto cubre el caso en que error es una Exception).
      final String errorStr = error.toString();
      // Busca el inicio del JSON buscando el primer '{'.
      final int jsonStart = errorStr.indexOf('{');
      if (jsonStart != -1) {
        final String jsonString = errorStr.substring(jsonStart);
        // Intenta decodificar el JSON.
        final dynamic errorData = jsonDecode(jsonString);

        // Si errorData es un Map, buscamos los mensajes.
        if (errorData is Map<String, dynamic>) {
          // Si existe la clave "message", la usamos.
          if (errorData.containsKey('message')) {
            return errorData['message'] ??
                'No se pudo enviar el enlace de recuperación.';
          }
          // Si existe la clave "errors" y es una lista, extraemos el "detail" del primer error.
          if (errorData.containsKey('errors') && errorData['errors'] is List) {
            final List errors = errorData['errors'];
            if (errors.isNotEmpty) {
              final firstError = errors.first;
              if (firstError is Map<String, dynamic> &&
                  firstError.containsKey('detail')) {
                return firstError['detail'] ??
                    'No se pudo enviar el enlace de recuperación.';
              }
            }
          }
        }
      }
      return 'No se pudo enviar el enlace de recuperación.';
    } catch (e) {
      return 'Error al procesar la respuesta del servidor';
    }
  }
}
