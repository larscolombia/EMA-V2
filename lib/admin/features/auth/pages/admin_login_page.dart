import 'package:ema_educacion_medica_avanzada/admin/features/auth/controllers/admin_auth_controller.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/background_widget.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/custom_text_field.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class AdminLoginPage extends StatelessWidget {
  const AdminLoginPage({super.key});

  @override
  Widget build(BuildContext context) {
    final controller = Get.put(AdminAuthController());

    return Scaffold(
      body: BackgroundWidget(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                // Logo arriba del card
                Image.asset('assets/images/logotype_white.png', height: 60),
                const SizedBox(height: 32),
                // Card de login
                Container(
                  constraints: const BoxConstraints(maxWidth: 450),
                  child: Card(
                    elevation: 8,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(24),
                    ),
                    child: Padding(
                      padding: const EdgeInsets.all(40),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          // Icono elegante con fondo circular
                          Container(
                            width: 80,
                            height: 80,
                            decoration: BoxDecoration(
                              gradient: LinearGradient(
                                begin: Alignment.topLeft,
                                end: Alignment.bottomRight,
                                colors: [
                                  AdminColors.primary,
                                  AdminColors.secondary,
                                ],
                              ),
                              shape: BoxShape.circle,
                              boxShadow: [
                                BoxShadow(
                                  color: AdminColors.primary.withValues(
                                    alpha: 0.3,
                                  ),
                                  blurRadius: 12,
                                  offset: const Offset(0, 4),
                                ),
                              ],
                            ),
                            child: const Icon(
                              Icons.admin_panel_settings_outlined,
                              size: 40,
                              color: Colors.white,
                            ),
                          ),
                          const SizedBox(height: 24),
                          Text(
                            'Panel Administrativo',
                            style: TextStyle(
                              fontSize: 24,
                              fontWeight: FontWeight.bold,
                              color: AdminColors.textPrimary,
                            ),
                          ),
                          const SizedBox(height: 8),
                          Text(
                            'EMA - Educación Médica Avanzada',
                            style: TextStyle(
                              fontSize: 14,
                              color: AdminColors.textSecondary,
                            ),
                          ),
                          const SizedBox(height: 32),
                          CustomTextField(
                            label: 'Correo electrónico',
                            controller: controller.emailController,
                            keyboardType: TextInputType.emailAddress,
                            textInputAction: TextInputAction.next,
                          ),
                          const SizedBox(height: 16),
                          CustomTextField(
                            label: 'Contraseña',
                            controller: controller.passwordController,
                            isPassword: true,
                            textInputAction: TextInputAction.done,
                          ),
                          const SizedBox(height: 12),
                          Obx(() {
                            if (controller.errorMessage.value.isNotEmpty) {
                              return Container(
                                padding: const EdgeInsets.all(12),
                                decoration: BoxDecoration(
                                  color: Colors.red[50],
                                  borderRadius: BorderRadius.circular(8),
                                  border: Border.all(color: Colors.red[200]!),
                                ),
                                child: Row(
                                  children: [
                                    Icon(
                                      Icons.error_outline,
                                      color: Colors.red[700],
                                    ),
                                    const SizedBox(width: 8),
                                    Expanded(
                                      child: Text(
                                        controller.errorMessage.value,
                                        style: TextStyle(
                                          color: Colors.red[700],
                                          fontSize: 13,
                                        ),
                                      ),
                                    ),
                                  ],
                                ),
                              );
                            }
                            return const SizedBox.shrink();
                          }),
                          const SizedBox(height: 24),
                          Obx(
                            () => SizedBox(
                              width: double.infinity,
                              height: 50,
                              child: ElevatedButton(
                                onPressed:
                                    controller.isLoading.value
                                        ? null
                                        : controller.login,
                                style: ElevatedButton.styleFrom(
                                  backgroundColor: AdminColors.primary,
                                  foregroundColor: Colors.white,
                                  shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(16),
                                  ),
                                  elevation: 2,
                                ),
                                child:
                                    controller.isLoading.value
                                        ? const SizedBox(
                                          height: 20,
                                          width: 20,
                                          child: CircularProgressIndicator(
                                            strokeWidth: 2,
                                            valueColor:
                                                AlwaysStoppedAnimation<Color>(
                                                  Colors.white,
                                                ),
                                          ),
                                        )
                                        : const Text(
                                          'Iniciar Sesión',
                                          style: TextStyle(
                                            fontSize: 16,
                                            fontWeight: FontWeight.bold,
                                          ),
                                        ),
                              ),
                            ),
                          ),
                          const SizedBox(height: 20),
                          Container(
                            padding: const EdgeInsets.all(12),
                            decoration: BoxDecoration(
                              color: Colors.amber[50],
                              borderRadius: BorderRadius.circular(8),
                              border: Border.all(color: Colors.amber[200]!),
                            ),
                            child: Row(
                              children: [
                                Icon(
                                  Icons.info_outline,
                                  color: Colors.amber[800],
                                  size: 20,
                                ),
                                const SizedBox(width: 8),
                                Expanded(
                                  child: Text(
                                    'Solo administradores pueden acceder',
                                    style: TextStyle(
                                      color: Colors.amber[900],
                                      fontSize: 12,
                                    ),
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
