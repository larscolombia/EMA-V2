import 'package:ema_educacion_medica_avanzada/core/auth/laravel_auth_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class ResetPasswordController extends GetxController {
  final newPasswordController = TextEditingController();
  final confirmPasswordController = TextEditingController();
  final LaravelAuthService _authService = Get.find<LaravelAuthService>();

  var isLoading = false.obs;
  var isSuccess = false.obs;
  String? resetToken;
  String? email;

  @override
  void onInit() {
    super.onInit();
    resetToken = Get.parameters['token'];
    email = Get.parameters['email'];

    if (resetToken == null ||
        resetToken!.isEmpty ||
        email == null ||
        email!.isEmpty) {
      Get.snackbar(
        'Error',
        'Token de recuperación inválido',
        backgroundColor: Colors.red.withValues(alpha: 0.8),
        colorText: Colors.white,
      );
      Future.delayed(const Duration(seconds: 2), () => Get.offNamed('/login'));
    }
  }

  Future<void> onResetPressed() async {
    final newPassword = newPasswordController.text.trim();
    final confirmPassword = confirmPasswordController.text.trim();

    if (newPassword.isEmpty || confirmPassword.isEmpty) {
      Get.snackbar(
        'Error',
        'Por favor complete todos los campos',
        backgroundColor: Colors.red.withValues(alpha: 0.8),
        colorText: Colors.white,
      );
      return;
    }

    if (newPassword != confirmPassword) {
      Get.snackbar(
        'Error',
        'Las contraseñas no coinciden',
        backgroundColor: Colors.red.withValues(alpha: 0.8),
        colorText: Colors.white,
      );
      return;
    }

    if (newPassword.length < 6) {
      Get.snackbar(
        'Error',
        'La contraseña debe tener al menos 6 caracteres',
        backgroundColor: Colors.red.withValues(alpha: 0.8),
        colorText: Colors.white,
      );
      return;
    }

    isLoading.value = true;

    try {
      await _authService.resetPassword(email!, resetToken!, newPassword);

      // Marcar como exitoso
      isSuccess.value = true;

      // Mostrar mensaje de éxito con duración más larga para web
      Get.snackbar(
        '✓ Contraseña actualizada',
        'Tu contraseña ha sido restablecida exitosamente. Ya puedes cerrar esta ventana e iniciar sesión en la app.',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.green.withValues(alpha: 0.9),
        colorText: Colors.white,
        duration: const Duration(seconds: 10),
        isDismissible: true,
      );

      // Limpiar los campos de contraseña
      newPasswordController.clear();
      confirmPasswordController.clear();

      // No redirigir automáticamente para permitir que el usuario cierre la ventana
      // Si está en móvil, podría abrir la app con deep linking, pero por ahora
      // dejamos que el usuario cierre manualmente
    } catch (e) {
      print('❌ ERROR RESET PASSWORD: $e');
      String errorMessage =
          'No se pudo restablecer la contraseña. Intenta nuevamente.';

      // Extract error message from exception
      if (e.toString().contains('Token inválido o expirado')) {
        errorMessage =
            'El token de recuperación ha expirado o es inválido. Solicita uno nuevo.';
      } else if (e.toString().contains('Usuario no encontrado')) {
        errorMessage = 'El usuario no fue encontrado.';
      }

      Get.snackbar(
        'Error',
        errorMessage,
        backgroundColor: Colors.red.withValues(alpha: 0.8),
        colorText: Colors.white,
        duration: const Duration(seconds: 5),
      );
    } finally {
      isLoading.value = false;
    }
  }

  @override
  void onClose() {
    newPasswordController.dispose();
    confirmPasswordController.dispose();
    super.onClose();
  }
}
