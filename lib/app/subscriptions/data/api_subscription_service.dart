import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:ema_educacion_medica_avanzada/app/subscriptions/subscriptions.dart';
import 'package:ema_educacion_medica_avanzada/config/constants/constants.dart';

class ApiSubscriptionService extends SubscriptionService {
  @override
  Future<List<Subscription>> fetchSubscriptions({
    required String authToken,
  }) async {
    try {
      final url = Uri.parse('$apiUrl/plans');
      final response = await http.get(
        url,
        headers: {
          'Authorization': 'Bearer $authToken',
          'Accept': 'application/json',
        },
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body)['data'] as List;
        return data.map((item) => Subscription.fromJson(item)).toList();
      } else {
        throw Exception(
          'Error al obtener las suscripciones: ${response.statusCode}',
        );
      }
    } catch (e) {
      throw Exception('Error al obtener las suscripciones: $e');
    }
  }

  @override
  Future<Subscription> createSubscription({
    required int userId,
    required int subscriptionPlanId,
    required int frequency,
    required String authToken,
  }) async {
    try {
      // Request checkout; return a placeholder while UI opens WebView
      await createCheckout(
        userId: userId,
        subscriptionPlanId: subscriptionPlanId,
        frequency: frequency,
        authToken: authToken,
      );
      return Subscription(
        id: 0,
        name: 'Procesando',
        currency: 'USD',
        price: 0,
        billing: 'Mensual',
        consultations: 0,
        questionnaires: 0,
        clinicalCases: 0,
        files: 0,
      );
    } catch (e) {
      throw Exception('Error al crear la suscripción: $e');
    }
  }

  Future<void> cancelSubscription({
    required int subscriptionId,
    required String authToken,
  }) async {
    try {
      final url = Uri.parse('$apiUrl/cancel-subscription');
      final response = await http.post(
        url,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $authToken',
        },
        body: jsonEncode({'subscription_id': subscriptionId}),
      );
      if (response.statusCode != 200 && response.statusCode != 201) {
        throw Exception(
          'Error al cancelar la suscripción: ${response.statusCode}',
        );
      }
    } catch (e) {
      throw Exception('Error al cancelar la suscripción: $e');
    }
  }

  @override
  Future<void> updateSubscriptionQuantities({
    required int subscriptionId,
    required String authToken,
    int? consultations,
    int? questionnaires,
    int? clinicalCases,
    int? files,
  }) async {
    try {
      final url = Uri.parse('$apiUrl/subscriptions/$subscriptionId');
      final Map<String, dynamic> body = {};

      if (consultations != null) body['consultations'] = consultations;
      if (questionnaires != null) body['questionnaires'] = questionnaires;
      if (clinicalCases != null) body['clinical_cases'] = clinicalCases;
      if (files != null) body['files'] = files;

      final response = await http.put(
        url,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $authToken',
        },
        body: jsonEncode(body),
      );

      if (response.statusCode != 200) {
        throw Exception(
          'Error al actualizar las cantidades de la suscripción: ${response.statusCode}',
        );
      }
    } catch (e) {
      throw Exception(
        'Error al actualizar las cantidades de la suscripción: $e',
      );
    }
  }

  // Create a checkout session and return the URL to open in WebView
  Future<String> createCheckout({
    required int userId,
    required int subscriptionPlanId,
    required int frequency,
    required String authToken,
  }) async {
    final url = Uri.parse('$apiUrl/checkout');
    final response = await http.post(
      url,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $authToken',
      },
      body: jsonEncode({
        'user_id': userId,
        'plan_id': subscriptionPlanId,
        'frequency': frequency,
      }),
    );
    if (response.statusCode != 200) {
      throw Exception('Error en checkout: ${response.statusCode}');
    }
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    final checkoutUrl = data['checkout_url']?.toString();
    if (checkoutUrl == null || checkoutUrl.isEmpty) {
      throw Exception('checkout_url no recibido');
    }
    return checkoutUrl;
  }
}
