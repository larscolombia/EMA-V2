import 'package:ema_educacion_medica_avanzada/admin/config/admin_routes.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/models/admin_user.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/services/admin_auth_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class AdminAuthController extends GetxController {
  final AdminAuthService _authService = Get.find<AdminAuthService>();

  final emailController = TextEditingController();
  final passwordController = TextEditingController();

  final isLoading = false.obs;
  final errorMessage = ''.obs;
  final currentUser = Rxn<AdminUser>();

  // Flag para evitar múltiples validaciones
  bool _hasCheckedAuth = false;

  @override
  void onInit() {
    super.onInit();
    // Solo validar si no se ha hecho antes
    if (!_hasCheckedAuth) {
      _hasCheckedAuth = true;
      _checkAuthentication();
    }
  }

  Future<void> _checkAuthentication() async {
    final user = await _authService.getCurrentUser();
    if (user != null) {
      currentUser.value = user;
      Get.offAllNamed(AdminRoutes.dashboard);
    }
  }

  Future<void> login() async {
    if (emailController.text.trim().isEmpty ||
        passwordController.text.trim().isEmpty) {
      errorMessage.value = 'Por favor complete todos los campos';
      return;
    }

    try {
      isLoading.value = true;
      errorMessage.value = '';

      final user = await _authService.login(
        emailController.text.trim().toLowerCase(),
        passwordController.text.trim(),
      );

      currentUser.value = user;
      Get.offAllNamed(AdminRoutes.dashboard);
    } catch (e) {
      errorMessage.value = e.toString().replaceAll('Exception: ', '');
    } finally {
      isLoading.value = false;
    }
  }

  Future<void> logout() async {
    try {
      await _authService.logout();
      currentUser.value = null;
      Get.offAllNamed(AdminRoutes.login);
    } catch (e) {
      print('Error al cerrar sesión: $e');
    }
  }

  @override
  void onClose() {
    emailController.dispose();
    passwordController.dispose();
    super.onClose();
  }
}
