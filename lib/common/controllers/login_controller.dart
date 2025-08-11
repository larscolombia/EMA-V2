import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/auth/laravel_auth_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:get/get.dart';

class LoginController extends GetxController {
  final LaravelAuthService _authService = Get.find<LaravelAuthService>();
  final UserService _userService = Get.find<UserService>();
  final FlutterSecureStorage _storage = const FlutterSecureStorage();

  // Controladores para los campos de texto
  TextEditingController emailController = TextEditingController();
  TextEditingController passwordController = TextEditingController();

  // Variable reactiva para el checkbox "Remember me"
  var rememberMe = false.obs;

  // Variable reactiva para el estado de carga
  var isLoading = false.obs;

  @override
  void onInit() {
    super.onInit();
    _loadRememberedCredentials();
  }

  // Método para cargar el email guardado, si existe
  Future<void> _loadRememberedCredentials() async {
    final rememberedEmail = await _storage.read(key: 'remembered_email');
    if (rememberedEmail != null) {
      emailController.text = rememberedEmail;
      rememberMe.value = true;
    }
  }

  // Método para actualizar el valor del checkbox
  void toggleRememberMe(bool? value) {
    rememberMe.value = value ?? false;
  }

  // Método para manejar el login
  Future<void> onLoginPressed() async {
    if (emailController.text.isEmpty || passwordController.text.isEmpty) {
      Get.snackbar(
        'Error',
        'Por favor, rellene todos los campos',
        backgroundColor: const Color.fromRGBO(244, 67, 54, 0.8),
        colorText: Colors.white,
      );
      return;
    }

    try {
      isLoading.value = true;
      final user = await _authService.login(
        emailController.text,
        passwordController.text,
      );

      await _storage.write(key: 'auth_token', value: user.authToken);
      await _storage.write(
          key: 'last_session', value: DateTime.now().toIso8601String());
      await _userService.setCurrentUser(user);

      // Manejar "Remember me" de forma más concisa
      if (rememberMe.value) {
        await _storage.write(
            key: 'remembered_email', value: emailController.text);
      } else {
        await _storage.delete(key: 'remembered_email');
      }

      Get.offAllNamed(Routes.home.name);
    } catch (e) {
      Get.snackbar(
        'Error de inicio de sesión',
        _extractErrorMessage(e),
        backgroundColor: const Color.fromRGBO(244, 67, 54, 0.8),
        colorText: Colors.white,
      );
    } finally {
      isLoading.value = false;
    }
  }

  String _extractErrorMessage(dynamic error) {
    try {
      final String errorStr = error.toString();
      final int jsonStart = errorStr.indexOf('{');
      if (jsonStart != -1) {
        final String jsonString = errorStr.substring(jsonStart);
        final dynamic errorData = jsonDecode(jsonString);

        if (errorData is Map<String, dynamic>) {
          if (errorData.containsKey('message')) {
            return errorData['message'] ?? 'Credenciales erróneas';
          }
          if (errorData.containsKey('errors') && errorData['errors'] is List) {
            final List errors = errorData['errors'];
            if (errors.isNotEmpty) {
              final firstError = errors.first;
              if (firstError is Map<String, dynamic> &&
                  firstError.containsKey('detail')) {
                return firstError['detail'] ?? 'Credenciales erróneas';
              }
            }
          }
        }
      }
      return 'Credenciales erróneas';
    } catch (e) {
      return 'Error al procesar la respuesta del servidor';
    }
  }

  @override
  void onClose() {
    emailController.dispose();
    passwordController.dispose();
    super.onClose();
  }
}
