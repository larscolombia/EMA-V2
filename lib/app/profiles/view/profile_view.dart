import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';

import '../widgets/premium_feature_wrapper.dart';

class ProfileView extends GetView<ProfileController> {
  const ProfileView({super.key});

  @override
  Widget build(BuildContext context) {
    final UserService userService = Get.find<UserService>();

    // Refrescar datos premium solo si tiene acceso a estadísticas
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (controller.currentProfile.value.activeSubscription?.statistics == 1 &&
          Get.find<UserTestProgressController>().testScores.isEmpty) {
        final progressController = Get.find<UserTestProgressController>();
        progressController.refreshAllStatistics(
          userId: controller.currentProfile.value.id,
          authToken: controller.currentProfile.value.authToken,
        );
      }
    });

    return Scaffold(
      appBar: AppBar(
        centerTitle: true,
        backgroundColor: AppStyles.primaryColor,
        leading: IconButton(
          icon: const Icon(Icons.logout),
          color: AppStyles.whiteColor,
          onPressed: () {
            Get.defaultDialog(
              title: 'Cerrar sesión',
              middleText: '¿Estás seguro de que deseas cerrar sesión?',
              textConfirm: 'Sí',
              textCancel: 'No',
              confirmTextColor: Colors.white,
              buttonColor: AppStyles.primaryColor,
              cancelTextColor: AppStyles.primaryColor,
              onConfirm: () {
                userService.logout();
                Get.back();
                Get.offAllNamed(Routes.login.name);
              },
            );
          },
        ),
        title: const Image(
          image: AssetImage('assets/images/logo.png'),
          height: 42,
          width: 120,
        ),
        actions: [
          IconButton(
            icon: AppIcons.closeSquare(
              height: 34,
              width: 34,
              color: AppStyles.whiteColor,
            ),
            onPressed: () {
              Get.toNamed(Routes.home.name);
            },
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: GetX<ProfileController>(
        builder: (controller) {
          if (controller.isLoading.value) {
            return const Center(child: CircularProgressIndicator());
          }

          final hasStatistics =
              controller.currentProfile.value.activeSubscription?.statistics ==
              1;

          return SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                ProfileHeader(
                  profile: controller.currentProfile.value,
                  isEditable: true,
                ),
                const SizedBox(height: 16),
                hasStatistics ? const PremiumBanner() : _buildBasicBanner(),
                const SizedBox(height: 16),
                PremiumFeatureWrapper(
                  isPremium: hasStatistics,
                  featureName: 'Estadísticas detalladas',
                  child: StatisticsSection(),
                ),
                const SizedBox(height: 24),
                ProfileInformation(
                  profile: controller.currentProfile.value,
                  countries: controller.countries,
                ),
                const SizedBox(height: 24),
                SubscriptionContainer(),
                const SizedBox(height: 15),
                AppScaffold.footerCredits(),
              ],
            ),
          );
        },
      ),
    );
  }

  Widget _buildBasicBanner() {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 5, horizontal: 12),
      margin: const EdgeInsets.symmetric(horizontal: 28),
      decoration: BoxDecoration(
        color: AppStyles.greyColor,
        borderRadius: BorderRadius.circular(8),
      ),
      child: const Row(
        children: [
          Expanded(
            child: Text(
              'Estás en el plan básico',
              style: TextStyle(
                color: AppStyles.whiteColor,
                fontSize: 15,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
