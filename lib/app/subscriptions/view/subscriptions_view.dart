import 'package:ema_educacion_medica_avanzada/app/subscriptions/subscriptions.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../common/widgets/app_scaffold.dart';

class SubscriptionsView extends StatelessWidget {
  const SubscriptionsView({super.key});

  @override
  Widget build(BuildContext context) {
    final SubscriptionController controller =
        Get.find<SubscriptionController>();

    return Scaffold(
      appBar: AppScaffold.appBar(
        forceBackRoute: true,
      ),
      body: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Suscripciones disponibles',
              style: TextStyle(
                color: AppStyles.primaryColor,
                fontSize: 22,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 16),
            Expanded(
              child: Obx(() {
                final subscriptions = controller.subscriptions;

                if (subscriptions.isEmpty) {
                  return const Center(child: CircularProgressIndicator());
                }

                final allSubscriptions = subscriptions;

                return ListView.separated(
                  itemCount: allSubscriptions.length,
                  separatorBuilder: (context, index) => Divider(
                    color: Colors.grey.shade300,
                    thickness: 1,
                    height: 32,
                  ),
                  itemBuilder: (context, index) {
                    final subscription = allSubscriptions[index];
                    return _buildSubscriptionCard(subscription, controller);
                  },
                );
              }),
            ),
          ],
        ),
      ),
      bottomNavigationBar: AppScaffold.footerCredits(),
    );
  }

  Widget _buildSubscriptionCard(
      Subscription subscription, SubscriptionController controller) {
    bool isFreePlan = subscription.name == 'Free';

    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(16),
        color: AppStyles.whiteColor,
        boxShadow: [
          BoxShadow(
            color: Colors.grey.withAlpha((0.2 * 255).toInt()),
            blurRadius: 12,
            offset: const Offset(0, 4),
          ),
        ],
        border: Border.all(
          color: isFreePlan ? Colors.green : AppStyles.primaryColor,
          width: 2,
        ),
      ),
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Nombre del plan
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Expanded(
                child: Text(
                  subscription.name,
                  textAlign: TextAlign.center,
                  softWrap: true,
                  overflow: TextOverflow.ellipsis,
                  maxLines: 2,
                  style: TextStyle(
                    color: AppStyles.primary900,
                    fontSize: 22,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
              if (isFreePlan)
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: Colors.green,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    'Gratis',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 14,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          // Precio y facturación
          RichText(
            text: TextSpan(
              text: '\$${subscription.price} ',
              style: TextStyle(
                color: AppStyles.primaryColor,
                fontSize: 25,
                fontWeight: FontWeight.bold,
              ),
              children: [
                TextSpan(
                  text:
                      '${subscription.currency.toUpperCase()} / ${subscription.billing}',
                  style: TextStyle(
                    color: Colors.grey.shade600,
                    fontSize: 16,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          // Beneficios
          _buildBenefitRow(
            'Restricción a ${subscription.consultations} consultas por mes',
            isFreePlan,
          ),
          _buildBenefitRow(
            'Restricción a ${subscription.questionnaires} cuestionarios',
            isFreePlan,
          ),
          _buildBenefitRow(
            'Restricción a ${subscription.clinicalCases} casos clínicos',
            isFreePlan,
          ),
          _buildBenefitRow(
            'Restricción a ${subscription.files} archivos',
            isFreePlan,
          ),
          const SizedBox(height: 16),
          // Botón para seleccionar plan
          Center(
            child: ElevatedButton(
              onPressed: () async {
                try {
                  final checkoutUrl = await controller.initiateCheckout(
                    subscriptionPlanId: subscription.id,
                    frequency: subscription.frequency ?? 0,
                  );
                  Get.to(() => StripeCheckoutView(checkoutUrl: checkoutUrl));
                } catch (e) {
                  Get.snackbar('Error', e.toString());
                }
              },
              style: ElevatedButton.styleFrom(
                backgroundColor: AppStyles.primary900,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
                padding:
                    const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
              ),
              child: Center(
                child: const Text(
                  'Seleccionar Plan',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    color: Colors.white,
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBenefitRow(String label, bool isFreePlan) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Icon(
            isFreePlan ? Icons.close_rounded : Icons.check_circle,
            color: isFreePlan ? Colors.red : AppStyles.primaryColor,
            size: 20,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              label,
              style: TextStyle(
                color: AppStyles.primaryColor,
                fontSize: 15,
                fontWeight: FontWeight.w600, // Resalta el texto
              ),
            ),
          ),
        ],
      ),
    );
  }
}
