import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class RegisterFormController extends GetxController {
  final LaravelAuthService _authService = Get.find<LaravelAuthService>();
  final CountryService _countryService = Get.find<CountryService>();

  @override
  void onInit() {
    super.onInit();
    loadCountries();
  }

  // Controladores para los campos obligatorios
  final emailController = TextEditingController();
  final firstNameController = TextEditingController();
  final lastNameController = TextEditingController();
  final passwordController = TextEditingController();
  final confirmPasswordController = TextEditingController();

  // Controladores para los campos opcionales
  final ageController = TextEditingController();
  final cityController = TextEditingController();
  final professionController = TextEditingController();

  // Variables reactivas para dropdowns
  var gender = ''.obs;

  RxList<CountryModel> countries = <CountryModel>[].obs;
  Rx<CountryModel?> selectedCountry = Rx<CountryModel?>(null);

  // Validaciones de la contraseña
  var hasMinLength = false.obs;
  var hasCapitalLetter = false.obs;
  var hasNumber = false.obs;

  // Estado del formulario en pasos
  var currentStep = 0.obs;

  // Estado de carga
  var isLoading = false.obs;

  /// Carga la lista de países desde el servicio
  Future<void> loadCountries() async {
    try {
      final fetchedCountries = await _countryService.getCountries();
      countries.value = fetchedCountries;

      // Selecciona el primer país como valor predeterminado
      if (fetchedCountries.isNotEmpty) {
        selectedCountry.value = fetchedCountries.first;
      }
    } catch (e) {
      Get.snackbar('Error', 'No se pudo cargar la lista de países: $e');
    }
  }

  /// Valida el formato del correo electrónico.
  String? validateEmail(String? value) {
    if (value == null || value.isEmpty) {
      return 'Por favor ingrese su correo electrónico';
    }
    if (!RegExp(r'^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$').hasMatch(value)) {
      return 'Por favor ingrese un correo electrónico válido';
    }
    return null;
  }

  /// Valida si un campo está vacío.
  String? validateField(String? value, String fieldName) {
    if (value == null || value.isEmpty) {
      return 'Por favor ingrese su $fieldName';
    }
    return null;
  }

  /// Valida la contraseña en tiempo real.
  void validatePassword(String value) {
    hasMinLength.value = value.length >= 8;
    hasCapitalLetter.value = value.contains(RegExp(r'[A-Z]'));
    hasNumber.value = value.contains(RegExp(r'\d'));
  }

  /// Avanza al siguiente paso del formulario.
  void nextStep() {
    currentStep.value++;
  }

  /// Retrocede al paso anterior del formulario.
  void previousStep() {
    currentStep.value--;
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
            return errorData['message'] ??
                'Error al registrar. Por favor intente de nuevo.';
          }
          if (errorData.containsKey('errors') && errorData['errors'] is List) {
            final List errors = errorData['errors'];
            if (errors.isNotEmpty) {
              final firstError = errors.first;
              if (firstError is Map<String, dynamic> &&
                  firstError.containsKey('detail')) {
                return firstError['detail'] ??
                    'Error al registrar. Por favor intente de nuevo.';
              }
            }
          }
        }
      }
      return 'Error al registrar. Por favor intente de nuevo.';
    } catch (e) {
      return 'Error al procesar la respuesta del servidor';
    }
  }

  /// Procesa el registro enviando los datos a la API.
  Future<void> onNextPressed() async {
    isLoading.value = true;

    // Validación de contraseña y confirmación
    if (passwordController.text != confirmPasswordController.text) {
      Get.snackbar(
        'Error',
        'Las contraseñas no coinciden',
        backgroundColor: const Color.fromRGBO(244, 67, 54, 0.8),
        colorText: Colors.white,
      );
      isLoading.value = false;
      return;
    }

    final formData = {
      'first_name': firstNameController.text,
      'last_name': lastNameController.text,
      'email': emailController.text,
      'password': passwordController.text,
      'password_confirmation': confirmPasswordController.text,
      'genero': gender.value.isEmpty ? null : gender.value,
      'edad': int.tryParse(ageController.text),
      'country_id': selectedCountry.value?.id,
      'city': cityController.text.isEmpty ? null : cityController.text,
      'profession':
          professionController.text.isEmpty ? null : professionController.text,
    };

    print('🚀 REGISTER INITIATED');
    print('Email: ${emailController.text}');
    print('API URL: $apiUrl');

    try {
      await _authService.register(formData);

      print('✅ Registration successful');

      // Si el registro es exitoso, redirigir al login
      Get.offNamed('/login');
      Get.snackbar(
        'Éxito',
        'Cuenta creada exitosamente',
        backgroundColor: const Color.fromRGBO(76, 175, 80, 0.8),
        colorText: Colors.white,
      );
    } catch (e) {
      print('❌ Registration failed: $e');
      // Manejo de errores del backend
      final errorMessage = _extractErrorMessage(e);
      Get.snackbar(
        'Error',
        errorMessage,
        backgroundColor: const Color.fromRGBO(244, 67, 54, 0.8),
        colorText: Colors.white,
      );
    } finally {
      isLoading.value = false;
    }
  }

  @override
  void onClose() {
    emailController.dispose();
    firstNameController.dispose();
    lastNameController.dispose();
    passwordController.dispose();
    confirmPasswordController.dispose();
    ageController.dispose();
    cityController.dispose();
    professionController.dispose();
    super.onClose();
  }
}
