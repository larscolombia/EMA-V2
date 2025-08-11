import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';

class PremiumFeatureWrapper extends StatelessWidget {
  final Widget child;
  final String featureName;
  final bool isPremium;

  const PremiumFeatureWrapper({
    super.key,
    required this.child,
    required this.featureName,
    required this.isPremium,
  });

  @override
  Widget build(BuildContext context) {
    if (isPremium) return child;

    return Stack(
      children: [
        AbsorbPointer(
          absorbing: true,
          child: Opacity(
            opacity: 0.35,
            child: child,
          ),
        ),
        Positioned.fill(
          child: Material(
            color: Colors.transparent,
            child: InkWell(
              onTap: () => _showPremiumDialog(context),
              child: Center(
                child: Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: Colors.black.withAlpha((0.6 * 255).toInt()),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        Icons.lock,
                        color: Colors.yellow[700],
                        size: 24,
                      ),
                      const SizedBox(width: 8),
                      Text(
                        'Función Premium',
                        style: TextStyle(
                          color: Colors.yellow[700],
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }

  void _showPremiumDialog(BuildContext context) {
    Get.dialog(
      AlertDialog(
        title: Row(
          children: [
            Icon(Icons.star, color: Colors.yellow[700]),
            const SizedBox(width: 8),
            const Text('Función Premium'),
          ],
        ),
        content:
            Text('$featureName está disponible solo para usuarios Premium.'),
        actions: [
          TextButton(
            onPressed: () => Get.back(),
            child: const Text('Cancelar'),
          ),
          ElevatedButton(
            onPressed: () {
              Get.back();
              Get.toNamed(Routes.subscriptions.name);
            },
            style: ElevatedButton.styleFrom(
              backgroundColor: AppStyles.primaryColor,
            ),
            child: const Text(
              'Actualizar a Premium',
              style: TextStyle(color: Colors.white),
            ),
          ),
        ],
      ),
    );
  }
}
