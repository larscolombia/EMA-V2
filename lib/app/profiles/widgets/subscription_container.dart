import 'package:flutter/material.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/data/api_subscription_service.dart';
import '../../profiles/controllers/profile_controller.dart';

class SubscriptionContainer extends StatelessWidget {
  const SubscriptionContainer({super.key});

  TextStyle _activePlanTitleStyle(BuildContext context) =>
      Theme.of(context).textTheme.titleLarge!.copyWith(
            color: AppStyles.primaryColor,
            fontWeight: FontWeight.bold,
          );

  TextStyle _planNameStyle(BuildContext context) =>
      Theme.of(context).textTheme.titleMedium!.copyWith(
            color: AppStyles.primary900,
            fontWeight: FontWeight.bold,
          );

  TextStyle _planCostStyle(BuildContext context) =>
      Theme.of(context).textTheme.titleLarge!.copyWith(
            color: AppStyles.primary900,
            fontWeight: FontWeight.bold,
            fontSize: 26, // Ajustamos ligeramente el tamaño del precio
            height: 1,
          );

  TextStyle _renewalTextStyle(BuildContext context) =>
      Theme.of(context).textTheme.bodySmall!.copyWith(
            color: AppStyles.greyColor,
          );

  TextStyle _cancelTextStyle(BuildContext context) =>
      Theme.of(context).textTheme.bodySmall!.copyWith(
            color: AppStyles.redColor,
            fontWeight: FontWeight.bold,
          );

  TextStyle _yearTextStyle(BuildContext context) =>
      Theme.of(context).textTheme.titleMedium!.copyWith(
            color: AppStyles.greyColor,
            fontSize: 16,
            height: 1,
          );

  @override
  Widget build(BuildContext context) {
    final profileController = Get.find<ProfileController>();

    return Obx(() {
      final profile = profileController.currentProfile.value;
      final activeSub = profile.activeSubscription;

      // Simplificamos la lógica - si no hay suscripción activa, es plan básico
      final bool isFree = activeSub == null;
      final String planName = isFree ? 'Plan Free' : activeSub.name;

      final String billing = activeSub != null
          ? (activeSub.frequency == 1
              ? "Mensual"
              : activeSub.frequency == 2
                  ? "Anual"
                  : activeSub.frequency == 3
                      ? "Pago Único"
                      : "M")
          : "M";
      final String renewalDate = isFree || activeSub.endDate == null
          ? ''
          : activeSub.endDate!.toString().split(' ')[0];

      final screenWidth = MediaQuery.of(context).size.width;
      final horizontalPadding = screenWidth < 360 ? 16.0 : 28.0;

      return Padding(
        padding: EdgeInsets.symmetric(
          horizontal: horizontalPadding,
          vertical: 16,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'SUSCRIPCIÓN ACTIVA',
              style: _activePlanTitleStyle(context),
            ),
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(24),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
                boxShadow: [
                  BoxShadow(
                    color: Colors.grey.withAlpha((0.2 * 255).toInt()),
                    blurRadius: 10,
                    offset: const Offset(0, 5),
                  ),
                ],
              ),
              child: LayoutBuilder(
                builder: (context, constraints) {
                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        planName,
                        style: _planNameStyle(context),
                      ),
                      const SizedBox(height: 2),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        crossAxisAlignment: CrossAxisAlignment.center,
                        children: [
                          Expanded(
                            flex: 3,
                            child: FittedBox(
                              fit: BoxFit.scaleDown,
                              alignment: Alignment.centerLeft,
                              child: Row(
                                crossAxisAlignment: CrossAxisAlignment.baseline,
                                textBaseline: TextBaseline.alphabetic,
                                children: [
                                  Text('\$', style: _planCostStyle(context)),
                                  const SizedBox(width: 2),
                                  Text(
                                    isFree ? '0' : activeSub.price.toString(),
                                    style: _planCostStyle(context),
                                  ),
                                  Padding(
                                    padding: const EdgeInsets.only(left: 4),
                                    child: FittedBox(
                                      fit: BoxFit.scaleDown,
                                      child: Text(
                                        '/ $billing',
                                        style: _yearTextStyle(context),
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                            ),
                          ),
                          const SizedBox(width: 16),
                          Flexible(
                            flex: 2,
                            child: _buildChangeButton(context, isFree),
                          ),
                        ],
                      ),
                      if (!isFree) ...[
                        const SizedBox(height: 16),
                        Center(
                          child: Text(
                            'Se renovará el $renewalDate',
                            style: _renewalTextStyle(context),
                          ),
                        )
                      ]
                    ],
                  );
                },
              ),
            ),
            if (!isFree) const SizedBox(height: 12),
            if (!isFree)
              Center(
                child: GestureDetector(
                  onTap: () async {
                    Get.defaultDialog(
                      title: 'Cancelar suscripción',
                      middleText: '¿Estás seguro de cancelar tu suscripción?',
                      textCancel: 'No',
                      textConfirm: 'Sí',
                      confirmTextColor: Colors.white,
                      buttonColor: AppStyles.primaryColor,
                      onConfirm: () async {
                        Get.back();
                        try {
                          Get.dialog(
                            const Center(child: CircularProgressIndicator()),
                            barrierDismissible: false,
                          );

                          await ApiSubscriptionService().cancelSubscription(
                            subscriptionId: activeSub.id,
                            authToken: profile.authToken,
                          );

                          // Forzar actualización usando forceCancel true
                          await profileController.refreshProfile(
                              forceCancel: true);
                          profileController.update();

                          Get.back();
                          Get.snackbar(
                            'Éxito',
                            'La suscripción fue cancelada',
                            backgroundColor:
                                Colors.green.withAlpha((0.8 * 255).toInt()),
                            colorText: Colors.white,
                            snackPosition: SnackPosition.TOP,
                          );
                        } catch (e) {
                          if (Get.isDialogOpen ?? false) {
                            Get.back();
                          }
                          Get.snackbar(
                            'Error',
                            e.toString(),
                            backgroundColor:
                                Colors.red.withAlpha((0.8 * 255).toInt()),
                            colorText: Colors.white,
                            snackPosition: SnackPosition.TOP,
                          );
                        }
                      },
                    );
                  },
                  child: Text(
                    'Cancelar suscripción',
                    style: _cancelTextStyle(context),
                  ),
                ),
              ),
          ],
        ),
      );
    });
  }

  Widget _buildChangeButton(BuildContext context, bool isFree) {
    final profileController = Get.find<ProfileController>();
    return SizedBox(
      height: 36,
      child: ElevatedButton(
        onPressed: () {
          Get.toNamed(Routes.subscriptions.name)?.then((_) {
            // Al volver de la renovación se refresca el perfil
            profileController.refreshProfile();
          });
        },
        style: ElevatedButton.styleFrom(
          backgroundColor: AppStyles.primary900,
          padding: const EdgeInsets.symmetric(horizontal: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(10),
          ),
          minimumSize: const Size(90, 36),
        ),
        child: Center(
          child: Text(
            'Cambiar',
            textAlign: TextAlign.center,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 14,
              fontWeight: FontWeight.w900,
            ),
          ),
        ),
      ),
    );
  }
}
