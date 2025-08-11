import 'package:ema_educacion_medica_avanzada/common/controllers.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/background_widget.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/custom_text_field.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/validation_bar.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class VerticalSpacing extends StatelessWidget {
  final double factor;
  const VerticalSpacing({super.key, this.factor = 1.0});

  @override
  Widget build(BuildContext context) {
    final screenHeight = MediaQuery.of(context).size.height;
    return SizedBox(height: screenHeight * 0.01 * factor);
  }
}

class WhiteCard extends StatelessWidget {
  final Widget child;
  final double padding;

  const WhiteCard({
    super.key,
    required this.child,
    required this.padding,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.all(padding),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(padding),
        boxShadow: [
          BoxShadow(
            color: Colors.grey.withAlpha((0.2 * 255).toInt()),
            blurRadius: 20,
            offset: const Offset(0, 10),
          ),
        ],
      ),
      child: child,
    );
  }
}

class RegisterFormView extends StatelessWidget {
  final RegisterFormController controller = Get.put(RegisterFormController());
  final _formKey = GlobalKey<FormState>();

  static const double horizontalPaddingFactor = 0.06;

  RegisterFormView({super.key});

  @override
  Widget build(BuildContext context) {
    final screenWidth = MediaQuery.of(context).size.width;
    final formPadding = screenWidth * horizontalPaddingFactor;

    return BackgroundWidget(
      child: SafeArea(
        child: Stack(
          children: [
            Padding(
              padding: EdgeInsets.all(formPadding),
              child: _RegisterFormContent(
                formKey: _formKey,
                controller: controller,
              ),
            ),
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

class _RegisterFormContent extends StatefulWidget {
  final GlobalKey<FormState> formKey;
  final RegisterFormController controller;

  const _RegisterFormContent({
    required this.formKey,
    required this.controller,
  });

  @override
  __RegisterFormContentState createState() => __RegisterFormContentState();
}

class __RegisterFormContentState extends State<_RegisterFormContent> {
  final _fieldFocusNodes = List<FocusNode>.generate(7, (index) => FocusNode());

  @override
  void dispose() {
    for (final focusNode in _fieldFocusNodes) {
      focusNode.dispose();
    }
    super.dispose();
  }

  bool _validateCurrentStep() {
    if (widget.controller.currentStep.value == 0) {
      return widget.formKey.currentState?.validate() ?? false;
    } else {
      return true;
    }
  }

  @override
  Widget build(BuildContext context) {
    final screenHeight = MediaQuery.of(context).size.height;
    final bottomInset = MediaQuery.of(context).viewInsets.bottom;

    return SingleChildScrollView(
      child: ConstrainedBox(
        constraints: BoxConstraints(minHeight: screenHeight),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _buildHeader(screenHeight),
            VerticalSpacing(factor: 2.0),
            Flexible(
              child: Padding(
                padding: EdgeInsets.only(bottom: bottomInset),
                child: WhiteCard(
                  padding: MediaQuery.of(context).size.width * 0.06,
                  child: SingleChildScrollView(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Form(
                          key: widget.formKey,
                          child: AnimatedSwitcher(
                            duration: const Duration(milliseconds: 300),
                            child: Obx(() {
                              return widget.controller.currentStep.value == 0
                                  ? StepOneForm(
                                      controller: widget.controller,
                                      focusNodes: _fieldFocusNodes,
                                    )
                                  : StepTwoForm(
                                      controller: widget.controller,
                                      focusNodes: _fieldFocusNodes,
                                    );
                            }),
                          ),
                        ),
                        VerticalSpacing(factor: 2.0),
                        Obx(() => NavigationButtons(
                              currentStep: widget.controller.currentStep.value,
                              onNext: () {
                                if (_validateCurrentStep()) {
                                  widget.controller.nextStep();
                                  _focusNextStep();
                                }
                              },
                              onBack: widget.controller.previousStep,
                              onSubmit: () {
                                if (_validateCurrentStep()) {
                                  widget.controller.onNextPressed();
                                }
                              },
                              isLoading: widget.controller.isLoading.value,
                            )),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _focusNextStep() {
    FocusScope.of(context).requestFocus(widget.controller.currentStep.value == 1
        ? _fieldFocusNodes[5]
        : _fieldFocusNodes[0]);
  }

  Widget _buildHeader(double screenHeight) {
    return Center(
      child: Image.asset(
        'assets/images/logotype_white.png',
        height: screenHeight * 0.1,
      ),
    );
  }
}

class StepOneForm extends StatelessWidget {
  final RegisterFormController controller;
  final List<FocusNode> focusNodes;

  const StepOneForm({
    super.key,
    required this.controller,
    required this.focusNodes,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        CustomTextField(
          label: 'Correo electrónico *',
          controller: controller.emailController,
          focusNode: focusNodes[0],
          keyboardType: TextInputType.emailAddress,
          nextFocusNode: focusNodes[1],
          textInputAction: TextInputAction.next,
          validator: controller.validateEmail,
        ),
        VerticalSpacing(),
        CustomTextField(
          label: 'Nombres *',
          controller: controller.firstNameController,
          focusNode: focusNodes[1],
          nextFocusNode: focusNodes[2],
          textInputAction: TextInputAction.next,
          validator: (value) => controller.validateField(value, 'nombre'),
        ),
        VerticalSpacing(),
        CustomTextField(
          label: 'Apellidos *',
          controller: controller.lastNameController,
          focusNode: focusNodes[2],
          nextFocusNode: focusNodes[3],
          textInputAction: TextInputAction.next,
          validator: (value) => controller.validateField(value, 'apellido'),
        ),
        VerticalSpacing(),
        CustomTextField(
          label: 'Contraseña *',
          controller: controller.passwordController,
          isPassword: true,
          focusNode: focusNodes[3],
          nextFocusNode: focusNodes[4],
          textInputAction: TextInputAction.next,
          onChanged: controller.validatePassword,
          validator: (value) {
            if (value == null || value.isEmpty) {
              return '¡Ups! Olvidaste ingresar tu contraseña.';
            }
            if (!controller.hasMinLength.value) {
              return 'Incluya al menos 8 caracteres.';
            }
            if (!controller.hasCapitalLetter.value) {
              return 'Incluya al menos 1 letra mayúscula.';
            }
            if (!controller.hasNumber.value) {
              return 'Incluya al menos 1 número.';
            }
            return null;
          },
        ),
        VerticalSpacing(),
        CustomTextField(
          label: 'Confirmar Contraseña *',
          controller: controller.confirmPasswordController,
          isPassword: true,
          focusNode: focusNodes[4],
          nextFocusNode: focusNodes[5],
          textInputAction: TextInputAction.next,
          validator: (value) {
            if (value == null || value.isEmpty) {
              return 'Por favor, confirme su contraseña';
            }
            if (value != controller.passwordController.text) {
              return 'Las contraseñas no coinciden';
            }
            return null;
          },
        ),
        VerticalSpacing(factor: 2.0),
        Obx(() => Row(
              mainAxisAlignment: MainAxisAlignment.spaceAround,
              children: [
                ValidationBar(
                  label: 'min 8 letras',
                  isValid: controller.hasMinLength.value,
                ),
                ValidationBar(
                  label: 'min 1 mayúscula',
                  isValid: controller.hasCapitalLetter.value,
                ),
                ValidationBar(
                  label: '1 numero',
                  isValid: controller.hasNumber.value,
                ),
              ],
            )),
      ],
    );
  }
}

class StepTwoForm extends StatelessWidget {
  final RegisterFormController controller;
  final List<FocusNode> focusNodes;

  const StepTwoForm({
    super.key,
    required this.controller,
    required this.focusNodes,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        DropdownButtonFormField<String>(
          decoration: InputDecoration(labelText: 'Genero'),
          items: ['Masculino', 'Femenino', 'Otro']
              .map((gender) => DropdownMenuItem(
                    value: gender,
                    child: Text(gender),
                  ))
              .toList(),
          onChanged: (value) => controller.gender.value = value ?? '',
        ),
        VerticalSpacing(),
        CustomTextField(
          label: 'Edad',
          controller: controller.ageController,
          focusNode: focusNodes[5],
          textInputAction: TextInputAction.next,
          keyboardType: TextInputType.number,
        ),
        VerticalSpacing(),
        // ComboBox de país
        Obx(() {
          return DropdownButtonFormField<CountryModel>(
            value: controller.selectedCountry.value,
            decoration: const InputDecoration(
              labelText: 'País',
            ),
            items: controller.countries.map((country) {
              return DropdownMenuItem<CountryModel>(
                value: country,
                child: Text(country.name),
              );
            }).toList(),
            onChanged: (value) {
              controller.selectedCountry.value = value;
            },
            isExpanded: true,
          );
        }),

        VerticalSpacing(),
        CustomTextField(
          label: 'Ciudad',
          controller: controller.cityController,
          focusNode: focusNodes[6],
          textInputAction: TextInputAction.next,
        ),
        VerticalSpacing(),
        CustomTextField(
          label: 'Profesión',
          controller: controller.professionController,
          textInputAction: TextInputAction.done,
        ),
      ],
    );
  }
}

class NavigationButtons extends StatelessWidget {
  final int currentStep;
  final VoidCallback onNext;
  final VoidCallback onBack;
  final VoidCallback onSubmit;
  final bool isLoading;

  const NavigationButtons({
    super.key,
    required this.currentStep,
    required this.onNext,
    required this.onBack,
    required this.onSubmit,
    required this.isLoading,
  });

  @override
  Widget build(BuildContext context) {
    final screenWidth = MediaQuery.of(context).size.width;
    final double baseFontSize = screenWidth * 0.035;

    if (currentStep == 0) {
      return Column(
        children: [
          SizedBox(
            width: double.infinity,
            child: ElevatedButton(
              onPressed: isLoading ? null : onNext,
              style: ElevatedButton.styleFrom(
                backgroundColor: AppStyles.primary900,
                padding: EdgeInsets.symmetric(vertical: baseFontSize * 0.25),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(16.0),
                ),
              ),
              child: isLoading
                  ? CircularProgressIndicator(color: Colors.white)
                  : Text(
                      'Siguiente',
                      style: TextStyle(
                        fontSize: baseFontSize,
                        color: Colors.white,
                      ),
                    ),
            ),
          ),
          VerticalSpacing(),
          TextButton(
            onPressed: isLoading ? null : () => Get.offNamed('/login'),
            child: RichText(
              text: TextSpan(
                text: '¿Tienes una cuenta? ',
                style: TextStyle(
                  fontSize: baseFontSize,
                  color: Colors.black,
                ),
                children: [
                  TextSpan(
                    text: 'Iniciar sesión',
                    style: TextStyle(
                      decoration: TextDecoration.underline,
                      fontWeight: FontWeight.bold,
                      fontSize: baseFontSize,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      );
    }

    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        ElevatedButton(
          onPressed: isLoading ? null : onBack,
          child: const Text('Atras'),
        ),
        ElevatedButton(
          onPressed: isLoading ? null : onSubmit,
          style: ElevatedButton.styleFrom(
            backgroundColor: AppStyles.primary900,
            padding:
                const EdgeInsets.symmetric(vertical: 12.0, horizontal: 24.0),
          ),
          child: const Text(
            'Registrarte',
            style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold),
          ),
        ),
      ],
    );
  }
}
