import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/app/profiles/models/most_studied_category.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/interfaces/subscription_service.dart';
import 'package:ema_educacion_medica_avanzada/config/constants/constants.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:image_picker/image_picker.dart';

class ProfileController extends GetxController {
  final UserService userService = Get.find<UserService>();
  final ProfileService profileService = Get.find<ProfileService>();
  final SubscriptionService subscriptionService =
      Get.find<SubscriptionService>();
  final CountryService _countryService = Get.find<CountryService>();

  Rx<UserModel> currentProfile = UserModel.unknow().obs;
  RxBool isLoading = true.obs;
  RxList<CountryModel> countries = <CountryModel>[].obs;
  final RxString selectedPlan = ''.obs;
  final user =
      UserModel(
        id: 0,
        firstName: '',
        lastName: '',
        email: '',
        status: false,
        language: '',
        darkMode: false,
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
        fullName: '',
        profileImage: '',
        authToken: '',
        countryName: '',
      ).obs;
  final Rx<MostStudiedCategory?> mostStudiedCategory =
      Rxn<MostStudiedCategory>();

  final RxInt remainingFiles = 0.obs;
  final RxInt remainingChats = 0.obs;
  final RxInt remainingClinicalCases = 0.obs;
  final RxInt remainingQuizzes = 0.obs;

  UserModel mergeProfiles(UserModel basic, UserModel detailed) {
    return basic.copyWith(
      firstName:
          detailed.firstName.isNotEmpty ? detailed.firstName : basic.firstName,
      lastName:
          detailed.lastName.isNotEmpty ? detailed.lastName : basic.lastName,
      email: detailed.email.isNotEmpty ? detailed.email : basic.email,
      profession: detailed.profession ?? basic.profession,
      gender: detailed.gender ?? basic.gender,
      age: detailed.age ?? basic.age,
      city: detailed.city ?? basic.city,
      countryName:
          detailed.countryName?.isNotEmpty == true
              ? detailed.countryName
              : basic.countryName,
      profileImage:
          detailed.profileImage.isNotEmpty
              ? detailed.profileImage
              : basic.profileImage,
      subscription: detailed.activeSubscription ?? basic.activeSubscription,
      authToken: basic.authToken,
    );
  }

  @override
  void onInit() {
    super.onInit();
    ever(currentProfile, (_) {
      _updateQuotas();
      update();
    });
    loadProfileData();
    loadCountries();
  }

  bool canUploadMoreFiles() {
    if (useAllFeatures) return true;
    final sub = currentProfile.value.activeSubscription;
    if (remainingFiles.value > 0) return true;
    // Fallback to subscription value in case quotas haven't been propagated yet
    return (sub?.files ?? 0) > 0;
  }

  bool canCreateMoreChats() {
    if (useAllFeatures) return true;
    final sub = currentProfile.value.activeSubscription;
    if (remainingChats.value > 0) return true;
    // Fallback to subscription value in case quotas haven't been propagated yet
    return (sub?.consultations ?? 0) > 0;
  }

  bool canCreateMoreClinicalCases() {
    return remainingClinicalCases.value > 0 || useAllFeatures;
  }

  bool canCreateMoreQuizzes() {
    return remainingQuizzes.value > 0 || useAllFeatures;
  }

  void updateSelectedPlan(String planName) {
    selectedPlan.value = planName;
  }

  Future<void> loadProfileData() async {
    try {
      isLoading.value = true;
      final basicProfile = userService.getProfileData();
      if (basicProfile.id != 0) {
        final detailedProfile = await profileService.fetchDetailedProfile(
          basicProfile,
        );

        // Fusionar datos para asegurarnos de conservar campos importantes
        final mergedProfile = mergeProfiles(basicProfile, detailedProfile);
        currentProfile.value = mergedProfile;
        await userService.setCurrentUser(mergedProfile);

        // Actualizar selectedPlan basado en statistics en lugar del nombre
        if (mergedProfile.activeSubscription?.statistics == 1) {
          selectedPlan.value = mergedProfile.activeSubscription!.name;
        } else {
          selectedPlan.value = 'Free';
        }

        update();
        Get.find<UserService>().update();
      }
    } catch (e) {
      final errorMessage = _extractErrorMessage(e);
      Get.snackbar(
        'Error',
        'Error al cargar el perfil: $errorMessage',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red.withAlpha((0.8 * 255).toInt()),
        colorText: Colors.white,
      );
    } finally {
      isLoading.value = false;
    }
  }

  Future<void> loadCountries() async {
    try {
      final fetchedCountries = await _countryService.getCountries();
      countries.value = fetchedCountries;
    } catch (e) {
      final errorMessage = _extractErrorMessage(e);
      Get.snackbar(
        'Error',
        'Error al cargar los países: $errorMessage',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red.withAlpha((0.8 * 255).toInt()),
        colorText: Colors.white,
      );
    }
  }

  Future<bool> updateProfile(UserModel updatedProfile) async {
    try {
      Get.dialog(
        const Center(child: CircularProgressIndicator()),
        barrierDismissible: false,
      );

      // Preservar datos importantes
      final currentToken = currentProfile.value.authToken;

      // Actualizar perfil
      final newProfile = await profileService.updateProfile(updatedProfile);

      // Usar merge para preservar datos críticos
      final mergedProfile = mergeProfiles(
        currentProfile.value,
        newProfile,
      ).copyWith(authToken: currentToken);
      currentProfile.value = mergedProfile;

      await userService.setCurrentUser(mergedProfile);

      update();
      Get.find<UserService>().update();

      await refreshProfile();

      Get.back();
      Get.back();

      Get.snackbar(
        'Éxito',
        'Perfil actualizado con éxito',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.green.withAlpha((0.8 * 255).toInt()),
        colorText: Colors.white,
      );

      return true;
    } catch (e) {
      Get.back();
      final errorMessage = _extractErrorMessage(e);
      Get.snackbar(
        'Error',
        'Error al actualizar el perfil: $errorMessage',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red.withAlpha((0.8 * 255).toInt()),
        colorText: Colors.white,
      );
      return false;
    }
  }

  Future<void> updateProfileImage(XFile imageFile) async {
    try {
      print(
        '🔄 [ProfileController] Iniciando actualización de imagen de perfil',
      );
      print('📁 [ProfileController] Archivo seleccionado: ${imageFile.path}');

      Get.dialog(
        const Center(child: CircularProgressIndicator()),
        barrierDismissible: false,
      );

      print('⏳ [ProfileController] Llamando al servicio de actualización...');
      final tempProfile = await profileService.updateProfileImage(
        currentProfile.value,
        imageFile,
      );
      print('✅ [ProfileController] Servicio completado exitosamente');
      print(
        '🖼️ [ProfileController] Nueva URL de imagen: ${tempProfile.profileImage}',
      );

      currentProfile.value = currentProfile.value.copyWith(
        profileImage: tempProfile.profileImage,
        authToken: currentProfile.value.authToken,
      );
      update();
      print('🔄 [ProfileController] Perfil actualizado localmente');

      print('🔄 [ProfileController] Refrescando perfil desde servidor...');
      final updatedProfile = await profileService.fetchDetailedProfile(
        currentProfile.value,
      );
      currentProfile.value = updatedProfile;
      update();
      print('✅ [ProfileController] Perfil refrescado exitosamente');

      Get.back();
      Get.snackbar(
        'Éxito',
        'Imagen actualizada con éxito',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.green.withAlpha((0.8 * 255).toInt()),
        colorText: Colors.white,
      );
      print('🎉 [ProfileController] Actualización completada con éxito');
    } catch (e) {
      print('❌ [ProfileController] Error en updateProfileImage: $e');
      print('📋 [ProfileController] Tipo de error: ${e.runtimeType}');

      Get.back();
      final errorMessage = _extractErrorMessage(e);
      print('📝 [ProfileController] Mensaje de error extraído: $errorMessage');

      Get.snackbar(
        'Error',
        errorMessage,
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red.withAlpha((0.8 * 255).toInt()),
        colorText: Colors.white,
      );
    }
  }

  Future<void> refreshProfile({bool forceCancel = false}) async {
    try {
      final fetchedProfile = await profileService.fetchDetailedProfile(
        currentProfile.value,
      );
      final updatedProfile =
          forceCancel
              ? fetchedProfile.copyWith(subscription: null)
              : fetchedProfile;

      currentProfile.value = updatedProfile;
      await userService.setCurrentUser(updatedProfile);
      _updateQuotas();

      // Solo actualizamos el nombre del plan
      selectedPlan.value = updatedProfile.activeSubscription?.name ?? 'Free';

      update();
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudo refrescar el perfil',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red,
        colorText: Colors.white,
      );
    }
  }

  Future<bool> decrementChatQuota() async {
    try {
      final subscription = currentProfile.value.activeSubscription;
      if (subscription == null) return false;

      await subscriptionService.updateSubscriptionQuantities(
        subscriptionId: subscription.id,
        authToken: currentProfile.value.authToken,
        consultations: 1, // Decrementar 1 chat
      );

      // Actualizar el perfil para obtener la nueva cuota
      await refreshProfile();
      return true;
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudo actualizar la cuota de chats',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red,
        colorText: Colors.white,
      );
      return false;
    }
  }

  Future<bool> decrementFileQuota() async {
    try {
      final subscription = currentProfile.value.activeSubscription;
      if (subscription == null) return false;

      await subscriptionService.updateSubscriptionQuantities(
        subscriptionId: subscription.id,
        authToken: currentProfile.value.authToken,
        files: 1,
      );

      // Actualizar el perfil para obtener la nueva cuota
      await refreshProfile();
      return true;
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudo actualizar la cuota de archivos',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red,
        colorText: Colors.white,
      );
      return false;
    }
  }

  void refreshFileQuota() {
    update();
  }

  void refreshChatQuota() {
    update();
  }

  Future<bool> decrementClinicalCaseQuota() async {
    try {
      final subscription = currentProfile.value.activeSubscription;
      if (subscription == null) return false;

      await subscriptionService.updateSubscriptionQuantities(
        subscriptionId: subscription.id,
        authToken: currentProfile.value.authToken,
        clinicalCases: 1, // Decrementar 1 caso clínico
      );

      // Actualizar el perfil para obtener la nueva cuota
      await refreshProfile();
      return true;
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudo actualizar la cuota de casos clínicos',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red,
        colorText: Colors.white,
      );
      return false;
    }
  }

  void refreshClinicalCaseQuota() {
    update();
  }

  Future<bool> decrementQuizQuota() async {
    try {
      final subscription = currentProfile.value.activeSubscription;
      if (subscription == null) return false;

      await subscriptionService.updateSubscriptionQuantities(
        subscriptionId: subscription.id,
        authToken: currentProfile.value.authToken,
        questionnaires: 1, // Decrementar 1 cuestionario
      );

      // Actualizar el perfil para obtener la nueva cuota
      await refreshProfile();
      return true;
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudo actualizar la cuota de cuestionarios',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red,
        colorText: Colors.white,
      );
      return false;
    }
  }

  void refreshQuizQuota() {
    update();
  }

  void _updateQuotas() {
    final subscription = currentProfile.value.activeSubscription;
    remainingFiles.value = subscription?.files ?? 0;
    remainingChats.value = subscription?.consultations ?? 0;
    remainingClinicalCases.value = subscription?.clinicalCases ?? 0;
    remainingQuizzes.value = subscription?.questionnaires ?? 0;
  }

  String _extractErrorMessage(dynamic error) {
    const genericMessage = "Ha ocurrido un error. Por favor intente de nuevo.";
    try {
      String errorStr = error.toString();
      if (errorStr.startsWith('Exception: ')) {
        errorStr = errorStr.substring('Exception: '.length);
      }
      if (errorStr.startsWith('{')) {
        final Map<String, dynamic> errorData = jsonDecode(errorStr);
        if (errorData.containsKey('message') &&
            errorData['message'].toString().isNotEmpty) {
          return errorData['message'];
        }
        if (errorData.containsKey('errors')) {
          final errors = errorData['errors'];
          if (errors is List && errors.isNotEmpty) {
            final msg = errors.join('. ');
            if (msg.isNotEmpty) return msg;
          }
          if (errors is Map && errors.isNotEmpty) {
            final msg = errors.values
                .expand((e) => e is List ? e : [e.toString()])
                .join('. ');
            if (msg.isNotEmpty) return msg;
          }
        }
      }
      return errorStr.isNotEmpty ? errorStr : genericMessage;
    } catch (e) {
      return genericMessage;
    }
  }
}
