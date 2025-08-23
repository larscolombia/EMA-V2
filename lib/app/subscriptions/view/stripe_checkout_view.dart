import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';
import 'package:get/get.dart';
import 'dart:convert';
import 'package:http/http.dart' as http;
import '../../../config/config.dart';
import '../../profiles/controllers/profile_controller.dart';

class StripeCheckoutView extends StatelessWidget {
  final String checkoutUrl;
  final String sessionId; // puede ser vacío en planes gratis

  const StripeCheckoutView({super.key, required this.checkoutUrl, required this.sessionId});

  @override
  Widget build(BuildContext context) {
    final profileController = Get.find<ProfileController>();
    final controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageFinished: (String url) async {
            if (url.contains('success')) {
              // Intentar confirmación manual (idempotente) si hay sessionId
              if (sessionId.isNotEmpty) {
                try {
                  final resp = await http.post(
                    Uri.parse('$apiUrl/stripe/confirm'),
                    headers: {
                      'Content-Type': 'application/json',
                      'Authorization': 'Bearer ${profileController.currentProfile.value.authToken}',
                    },
                    body: jsonEncode({'session_id': sessionId}),
                  );
                  // Ignorar errores silenciosamente; se refresca igual
                  debugPrint('Confirm resp: ${resp.statusCode} ${resp.body}');
                } catch (_) {}
              }
              await profileController.refreshProfile();
              Get.offAllNamed(Routes.profile.name);
            }
          },
        ),
      )
      ..loadRequest(Uri.parse(checkoutUrl));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Pago con Stripe'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () async {
            // Al volver manualmente, intentar refrescar perfil para captar subs auto creada
            await profileController.refreshProfile();
            Get.offAllNamed(Routes.profile.name);
          },
        ),
      ),
      body: WebViewWidget(controller: controller),
    );
  }
}
