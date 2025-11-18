import 'package:ema_educacion_medica_avanzada/common/controllers/reset_password_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/background_widget.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/custom_text_field.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class ResetPasswordScreen extends StatelessWidget {
  ResetPasswordScreen({super.key});

  final ResetPasswordController controller = Get.put(ResetPasswordController());
  final _formKey = GlobalKey<FormState>();

  @override
  Widget build(BuildContext context) {
    final isWeb = MediaQuery.of(context).size.width > 600;

    return BackgroundWidget(
      child: SafeArea(
        child: Stack(
          children: [
            Center(
              child: SingleChildScrollView(
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 24,
                    vertical: 40,
                  ),
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 450),
                    child: Form(
                      key: _formKey,
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          // Logo
                          Center(
                            child: Image.asset(
                              'assets/images/logotype_white.png',
                              height: isWeb ? 80 : 60,
                            ),
                          ),
                          const SizedBox(height: 40),

                          // Tarjeta blanca
                          Container(
                            padding: EdgeInsets.all(isWeb ? 40 : 32),
                            decoration: BoxDecoration(
                              color: Colors.white,
                              borderRadius: BorderRadius.circular(24.0),
                              boxShadow: [
                                BoxShadow(
                                  color: Colors.black.withAlpha(
                                    (0.15 * 255).toInt(),
                                  ),
                                  blurRadius: 20,
                                  offset: const Offset(0, 10),
                                ),
                              ],
                            ),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.stretch,
                              children: [
                                // Mostrar mensaje de éxito o formulario
                                Obx(() {
                                  if (controller.isSuccess.value) {
                                    return Column(
                                      children: [
                                        const Icon(
                                          Icons.check_circle,
                                          color: Colors.green,
                                          size: 80,
                                        ),
                                        const SizedBox(height: 24),
                                        const Text(
                                          '¡Contraseña actualizada!',
                                          textAlign: TextAlign.center,
                                          style: TextStyle(
                                            fontSize: 24,
                                            fontWeight: FontWeight.bold,
                                            color: Colors.green,
                                          ),
                                        ),
                                        const SizedBox(height: 16),
                                        Text(
                                          'Tu contraseña ha sido restablecida exitosamente.',
                                          textAlign: TextAlign.center,
                                          style: TextStyle(
                                            fontSize: 16,
                                            color: Colors.grey[700],
                                          ),
                                        ),
                                        const SizedBox(height: 12),
                                        Text(
                                          'Ya puedes cerrar esta ventana e iniciar sesión en la app con tu nueva contraseña.',
                                          textAlign: TextAlign.center,
                                          style: TextStyle(
                                            fontSize: 14,
                                            color: Colors.grey[600],
                                          ),
                                        ),
                                      ],
                                    );
                                  }

                                  // Formulario normal
                                  return Column(
                                    crossAxisAlignment:
                                        CrossAxisAlignment.stretch,
                                    children: [
                                      // Título
                                      Text(
                                        'Restablecer Contraseña',
                                        textAlign: TextAlign.center,
                                        style: TextStyle(
                                          fontSize: isWeb ? 24 : 20,
                                          fontWeight: FontWeight.bold,
                                          color: AppStyles.primary900,
                                        ),
                                      ),
                                      const SizedBox(height: 12),

                                      // Descripción
                                      Text(
                                        'Ingresa tu nueva contraseña',
                                        textAlign: TextAlign.center,
                                        style: TextStyle(
                                          fontSize: 14,
                                          color: Colors.grey[600],
                                        ),
                                      ),
                                      const SizedBox(height: 32),

                                      // Campo Nueva Contraseña
                                      CustomTextField(
                                        label: 'Nueva Contraseña',
                                        controller:
                                            controller.newPasswordController,
                                        isPassword: true,
                                        validator: (value) {
                                          if (value == null || value.isEmpty) {
                                            return 'Por favor ingrese su nueva contraseña';
                                          }
                                          if (value.length < 6) {
                                            return 'La contraseña debe tener al menos 6 caracteres';
                                          }
                                          return null;
                                        },
                                      ),
                                      const SizedBox(height: 20),

                                      // Campo Confirmar Contraseña
                                      CustomTextField(
                                        label: 'Confirmar Contraseña',
                                        controller:
                                            controller
                                                .confirmPasswordController,
                                        isPassword: true,
                                        validator: (value) {
                                          if (value == null || value.isEmpty) {
                                            return 'Por favor confirme su contraseña';
                                          }
                                          if (value !=
                                              controller
                                                  .newPasswordController
                                                  .text) {
                                            return 'Las contraseñas no coinciden';
                                          }
                                          return null;
                                        },
                                      ),
                                      const SizedBox(height: 32),

                                      // Botón Restablecer
                                      Obx(() {
                                        return ElevatedButton(
                                          onPressed:
                                              controller.isLoading.value
                                                  ? null
                                                  : () {
                                                    if (_formKey.currentState!
                                                        .validate()) {
                                                      controller
                                                          .onResetPressed();
                                                    }
                                                  },
                                          style: ElevatedButton.styleFrom(
                                            backgroundColor:
                                                AppStyles.primary900,
                                            padding: EdgeInsets.symmetric(
                                              vertical: isWeb ? 18 : 16,
                                            ),
                                            shape: RoundedRectangleBorder(
                                              borderRadius:
                                                  BorderRadius.circular(16.0),
                                            ),
                                          ),
                                          child:
                                              controller.isLoading.value
                                                  ? const SizedBox(
                                                    height: 20,
                                                    width: 20,
                                                    child:
                                                        CircularProgressIndicator(
                                                          color: Colors.white,
                                                          strokeWidth: 2,
                                                        ),
                                                  )
                                                  : Text(
                                                    'Restablecer Contraseña',
                                                    style: TextStyle(
                                                      fontSize: isWeb ? 16 : 15,
                                                      color: Colors.white,
                                                      fontWeight:
                                                          FontWeight.w600,
                                                    ),
                                                  ),
                                        );
                                      }),
                                      const SizedBox(height: 20),

                                      // Botón volver a login
                                      Center(
                                        child: TextButton(
                                          onPressed:
                                              () => Get.offNamed('/login'),
                                          child: RichText(
                                            text: TextSpan(
                                              text: 'Volver a ',
                                              style: TextStyle(
                                                fontSize: 14,
                                                color: Colors.grey[700],
                                              ),
                                              children: const [
                                                TextSpan(
                                                  text: 'Iniciar sesión',
                                                  style: TextStyle(
                                                    decoration:
                                                        TextDecoration
                                                            .underline,
                                                    fontWeight: FontWeight.bold,
                                                    color: AppStyles.primary900,
                                                  ),
                                                ),
                                              ],
                                            ),
                                          ),
                                        ),
                                      ),
                                    ],
                                  );
                                }),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),

            // Loading overlay
            Obx(() {
              if (controller.isLoading.value) {
                return Container(
                  color: Colors.black.withAlpha((0.5 * 255).toInt()),
                  child: const Center(
                    child: CircularProgressIndicator(
                      color: AppStyles.primary900,
                    ),
                  ),
                );
              }
              return const SizedBox.shrink();
            }),
          ],
        ),
      ),
    );
  }
}
