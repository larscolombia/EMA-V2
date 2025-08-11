import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';
import 'package:get/get.dart';
import '../../../config/config.dart';
import '../../profiles/controllers/profile_controller.dart';

class StripeCheckoutView extends StatelessWidget {
  final String checkoutUrl;

  const StripeCheckoutView({super.key, required this.checkoutUrl});

  @override
  Widget build(BuildContext context) {
    final controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageFinished: (String url) async {
            if (url.contains('success')) {
              await Get.find<ProfileController>().refreshProfile();
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
          onPressed: () => Get.offAllNamed(Routes.profile.name),
        ),
      ),
      body: WebViewWidget(controller: controller),
    );
  }
}
