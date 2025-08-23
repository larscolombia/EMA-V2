import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/subscriptions.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/view/stripe_checkout_view.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'dart:convert';

class SubscriptionController extends GetxController {
  final RxList<Subscription> subscriptions = <Subscription>[].obs;
  final RxInt activePlanId = 0.obs;
  final ApiSubscriptionService _apiSubscriptionService =
      ApiSubscriptionService();
  final UserService _userService = Get.find<UserService>();

  var isLoading = false.obs;

  @override
  void onInit() {
    super.onInit();
    fetchSubscriptions();
  }

  Future<void> fetchSubscriptions() async {
    try {
      isLoading.value = true;
      final currentUser = _userService.getProfileData();
      final fetchedSubscriptions = await _apiSubscriptionService
          .fetchSubscriptions(authToken: currentUser.authToken);
  subscriptions.value = fetchedSubscriptions;
  // Detect active plan
  final active = fetchedSubscriptions.firstWhereOrNull((s) => s.active);
  activePlanId.value = active?.id ?? 0;
    } catch (e) {
      final errorMessage = _extractErrorMessage(e);
      Get.snackbar(
        'Error',
        errorMessage,
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red.withAlpha((0.8 * 255).toInt()),
        colorText: Colors.white,
      );
    } finally {
      isLoading.value = false;
    }
  }

  Future<void> createSubscription({
    required int subscriptionPlanId,
    required int frequency,
  }) async {
    try {
      if (activePlanId.value != 0 && activePlanId.value == subscriptionPlanId) {
        Get.snackbar(
          'Plan activo',
          'Este ya es tu plan actual.',
          snackPosition: SnackPosition.TOP,
          backgroundColor: Colors.blue.withAlpha((0.85 * 255).toInt()),
          colorText: Colors.white,
        );
        return;
      }
      final currentUser = _userService.getProfileData();
  final result = await _apiSubscriptionService.createCheckoutFull(
        userId: currentUser.id,
        subscriptionPlanId: subscriptionPlanId,
        frequency: frequency,
        authToken: currentUser.authToken,
      );
  final checkoutUrl = result.url;
  final sessionId = result.sessionId;
  final autoSubscribed = result.autoSubscribed;
  if (autoSubscribed) {
        await fetchSubscriptions();
        Get.snackbar(
          'Éxito',
          'Suscripción actualizada',
          snackPosition: SnackPosition.TOP,
          backgroundColor: Colors.green.withAlpha((0.85 * 255).toInt()),
          colorText: Colors.white,
        );
        return;
      }
  Get.to(() => StripeCheckoutView(checkoutUrl: checkoutUrl, sessionId: sessionId));
    } catch (e) {
      final errorMessage = _extractErrorMessage(e);
      Get.snackbar(
        'Error',
        errorMessage,
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.red.withAlpha((0.8 * 255).toInt()),
        colorText: Colors.white,
      );
      throw Exception(errorMessage);
    }
  }


  // Centralizado para manejar errores de cuota agotada (403) en otras capas.
  void handleQuotaExceeded() {
    Get.snackbar(
      'Plan agotado',
      'Has alcanzado el límite de tu plan. Elige un plan superior.',
      snackPosition: SnackPosition.TOP,
      backgroundColor: Colors.orange.withAlpha((0.9 * 255).toInt()),
      colorText: Colors.white,
      duration: const Duration(seconds: 4),
    );
    // Navega a la vista de planes si está registrada
    if (Get.isRegistered<SubscriptionController>()) {
      // Si ya estamos en esta vista no duplicar navegación
      Future.delayed(const Duration(milliseconds: 400), () {
        if (Get.currentRoute != '/subscriptions') {
          Get.toNamed('/subscriptions');
        }
      });
    }
  }

  /// Extrae el mensaje de error del JSON de respuesta.
  String _extractErrorMessage(dynamic error) {
    try {
      // Convierte el error a String (esto cubre el caso en que error es una Exception).
      final String errorStr = error.toString();
      // Busca el inicio del JSON buscando el primer '{'.
      final int jsonStart = errorStr.indexOf('{');
      if (jsonStart != -1) {
        final String jsonString = errorStr.substring(jsonStart);
        // Intenta decodificar el JSON.
        final dynamic errorData = jsonDecode(jsonString);

        // Si errorData es un Map, buscamos los mensajes.
        if (errorData is Map<String, dynamic>) {
          // Si existe la clave "message", la usamos.
          if (errorData.containsKey('message')) {
            return errorData['message'] ?? 'Error al procesar la solicitud.';
          }
          // Si existe la clave "errors" y es una lista, extraemos el "detail" del primer error.
          if (errorData.containsKey('errors') && errorData['errors'] is List) {
            final List errors = errorData['errors'];
            if (errors.isNotEmpty) {
              final firstError = errors.first;
              if (firstError is Map<String, dynamic> &&
                  firstError.containsKey('detail')) {
                return firstError['detail'] ??
                    'Error al procesar la solicitud.';
              }
            }
          }
        }
      }
      return 'Error al procesar la solicitud.';
    } catch (e) {
      return 'Error al procesar la respuesta del servidor';
    }
  }
}
