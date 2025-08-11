import 'package:ema_educacion_medica_avanzada/common/controllers.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/background_widget.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/custom_text_field.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class ForgotPasswordScreen extends StatelessWidget {
  ForgotPasswordScreen({super.key});

  final ForgotPasswordController controller =
      Get.put(ForgotPasswordController());
  final _formKey = GlobalKey<FormState>();

  @override
  Widget build(BuildContext context) {
    final screenHeight = MediaQuery.of(context).size.height;
    final screenWidth = MediaQuery.of(context).size.width;
    // Definimos un tamaño base para esta vista
    final double baseFontSize = screenWidth * 0.035;
    final double logoHeight = screenHeight * 0.1;
    final double fieldSpacing = screenHeight * 0.02;
    final double cardPadding = screenWidth * 0.06;

    return BackgroundWidget(
      child: SafeArea(
        child: Stack(
          children: [
            SingleChildScrollView(
              child: SizedBox(
                height: screenHeight,
                child: Padding(
                  padding: EdgeInsets.all(cardPadding),
                  child: Form(
                    key: _formKey,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        // Logo de la aplicación
                        Center(
                          child: Image.asset(
                            'assets/images/logotype_white.png',
                            height: logoHeight,
                          ),
                        ),
                        SizedBox(height: fieldSpacing * 2),

                        // Tarjeta blanca con el formulario de Forgot Password
                        Center(
                          child: ConstrainedBox(
                            constraints: BoxConstraints(
                              maxWidth:
                                  screenWidth > 600 ? 400 : screenWidth * 0.9,
                            ),
                            child: Container(
                              padding: EdgeInsets.all(cardPadding),
                              decoration: BoxDecoration(
                                color: Colors.white,
                                borderRadius: BorderRadius.circular(24.0),
                                boxShadow: [
                                  BoxShadow(
                                    color: Colors.black
                                        .withAlpha((0.1 * 255).toInt()),
                                    blurRadius: 10,
                                    offset: const Offset(0, 5),
                                  ),
                                ],
                              ),
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  // Campo de texto para Email o Teléfono
                                  CustomTextField(
                                    label: 'Correo electrónico',
                                    controller: controller.emailController,
                                    validator: (value) {
                                      if (value == null || value.isEmpty) {
                                        return 'Por favor ingrese su correo electrónico';
                                      }
                                      return null;
                                    },
                                  ),
                                  SizedBox(height: fieldSpacing * 1.5),

                                  // Botón "Next"
                                  SizedBox(
                                    width: double.infinity,
                                    child: Obx(() {
                                      return ElevatedButton(
                                        onPressed: controller.isLoading.value
                                            ? null
                                            : () {
                                                if (_formKey.currentState!
                                                    .validate()) {
                                                  controller.onNextPressed();
                                                }
                                              },
                                        style: ElevatedButton.styleFrom(
                                          backgroundColor: AppStyles.primary900,
                                          padding: EdgeInsets.symmetric(
                                            vertical: screenHeight * 0.02,
                                          ),
                                          shape: RoundedRectangleBorder(
                                            borderRadius:
                                                BorderRadius.circular(16.0),
                                          ),
                                        ),
                                        child: Text(
                                          'Siguiente',
                                          style: TextStyle(
                                            fontSize: baseFontSize,
                                            color: Colors.white,
                                          ),
                                        ),
                                      );
                                    }),
                                  ),
                                  SizedBox(height: fieldSpacing),

                                  // Botón para regresar a la pantalla de login
                                  Center(
                                    child: TextButton(
                                      onPressed: () {
                                        Get.offNamed(
                                            '/login'); // Navegar a LoginScreen
                                      },
                                      child: RichText(
                                        text: TextSpan(
                                          text: 'Volver a ',
                                          style: TextStyle(
                                            fontSize: baseFontSize,
                                            color: Colors.black,
                                          ),
                                          children: [
                                            TextSpan(
                                              text: 'Iniciar sesión',
                                              style: TextStyle(
                                                decoration:
                                                    TextDecoration.underline,
                                                fontWeight: FontWeight.bold,
                                                fontSize: baseFontSize,
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
                        SizedBox(height: screenHeight * 0.1),
                      ],
                    ),
                  ),
                ),
              ),
            ),

            // Indicador de carga en toda la pantalla
            Obx(() {
              if (controller.isLoading.value) {
                return Container(
                  color: Colors.black.withAlpha((0.5 * 255).toInt()),
                  child: Center(
                    child:
                        CircularProgressIndicator(color: AppStyles.primary900),
                  ),
                );
              }
              return SizedBox.shrink();
            }),
          ],
        ),
      ),
    );
  }
}
