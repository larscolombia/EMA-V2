import 'package:ema_educacion_medica_avanzada/common/controllers/login_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/background_widget.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/custom_text_field.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class LoginScreen extends StatelessWidget {
  LoginScreen({super.key});

  final LoginController controller = Get.put(LoginController());
  final _formKey = GlobalKey<FormState>();

  @override
  Widget build(BuildContext context) {
    final screenHeight = MediaQuery.sizeOf(context).height;
    final screenWidth = MediaQuery.sizeOf(context).width;
    final isKeyboardVisible = MediaQuery.of(context).viewInsets.bottom > 0;

    final double logoHeight = screenHeight * 0.1;
    final double fieldSpacing = screenHeight * 0.01;
    final double cardPadding = screenWidth * 0.06;
    final double baseFontSize = screenWidth * 0.035;

    return Scaffold(
      resizeToAvoidBottomInset: true,
      body: Stack(
        children: [
          BackgroundWidget(
            child: SafeArea(
              child: LayoutBuilder(
                builder: (context, constraints) {
                  final availableHeight = constraints.maxHeight -
                      MediaQuery.of(context).viewInsets.bottom;
                  final loginContent = Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      // Logo (siempre visible)
                      Center(
                        child: Image.asset(
                          'assets/images/logotype_white.png',
                          height: logoHeight,
                        ),
                      ),
                      SizedBox(height: fieldSpacing * 2),
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
                            child: Form(
                              key: _formKey,
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.stretch,
                                children: [
                                  CustomTextField(
                                    label: 'Correo electrónico',
                                    keyboardType: TextInputType.emailAddress,
                                    controller: controller.emailController,
                                    validator: (value) {
                                      if (value == null || value.isEmpty) {
                                        return 'Por favor ingrese su correo electrónico';
                                      }
                                      return null;
                                    },
                                  ),
                                  SizedBox(height: fieldSpacing),
                                  CustomTextField(
                                    label: 'Contraseña',
                                    controller: controller.passwordController,
                                    isPassword: true,
                                    textInputAction: TextInputAction.done,
                                    validator: (value) {
                                      if (value == null || value.isEmpty) {
                                        return 'Por favor, introduzca su contraseña';
                                      }
                                      return null;
                                    },
                                  ),
                                  SizedBox(height: fieldSpacing / 10),
                                  Obx(
                                    () => Row(
                                      mainAxisAlignment:
                                          MainAxisAlignment.spaceBetween,
                                      crossAxisAlignment:
                                          CrossAxisAlignment.center,
                                      children: [
                                        Row(
                                          mainAxisSize: MainAxisSize.min,
                                          children: [
                                            Checkbox(
                                              value:
                                                  controller.rememberMe.value,
                                              onChanged:
                                                  controller.toggleRememberMe,
                                            ),
                                            Text(
                                              'Recordar',
                                              style: TextStyle(
                                                  fontSize: baseFontSize),
                                            ),
                                          ],
                                        ),
                                        Flexible(
                                          child: TextButton(
                                            onPressed: () {
                                              Get.toNamed('/forgot-password');
                                            },
                                            child: Text(
                                              '¿Olvidaste tu contraseña?',
                                              style: TextStyle(
                                                fontSize: baseFontSize,
                                                color: AppStyles.greyColor,
                                              ),
                                              overflow: TextOverflow.ellipsis,
                                            ),
                                          ),
                                        ),
                                      ],
                                    ),
                                  ),
                                  SizedBox(height: fieldSpacing * 1.5),
                                  SizedBox(
                                    width: double.infinity,
                                    child: ElevatedButton(
                                      onPressed: () {
                                        if (_formKey.currentState!.validate()) {
                                          controller.onLoginPressed();
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
                                        'Iniciar Sesión',
                                        style: TextStyle(
                                          fontSize: screenWidth * 0.045,
                                          color: Colors.white,
                                        ),
                                      ),
                                    ),
                                  ),
                                  SizedBox(height: fieldSpacing),
                                  Center(
                                    child: TextButton(
                                      onPressed: () {
                                        Get.toNamed('/register');
                                      },
                                      child: RichText(
                                        text: TextSpan(
                                          text: '¿No tienes una cuenta? ',
                                          style: TextStyle(
                                            fontSize: baseFontSize,
                                            color: Colors.black,
                                          ),
                                          children: [
                                            TextSpan(
                                              text: 'Regístrate',
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
                      ),
                    ],
                  );

                  final content = ConstrainedBox(
                    constraints: BoxConstraints(minHeight: availableHeight),
                    child: IntrinsicHeight(child: loginContent),
                  );

                  if (isKeyboardVisible) {
                    return SingleChildScrollView(
                      physics: ClampingScrollPhysics(),
                      padding: EdgeInsets.only(
                        left: cardPadding,
                        right: cardPadding,
                        top: cardPadding,
                        bottom: MediaQuery.of(context).viewInsets.bottom + 20,
                      ),
                      child: content,
                    );
                  } else {
                    // Se agrega un padding superior para bajar la tarjeta
                    return Center(
                      child: Padding(
                        padding: EdgeInsets.only(
                            top: 50), // Ajusta este valor según prefieras
                        child: content,
                      ),
                    );
                  }
                },
              ),
            ),
          ),
          Obx(
            () => controller.isLoading.value
                ? Container(
                    color: Colors.black.withAlpha((0.5 * 255).toInt()),
                    child: Center(
                      child: CircularProgressIndicator(
                        valueColor:
                            AlwaysStoppedAnimation<Color>(AppStyles.primary900),
                      ),
                    ),
                  )
                : const SizedBox.shrink(),
          ),
        ],
      ),
    );
  }
}
