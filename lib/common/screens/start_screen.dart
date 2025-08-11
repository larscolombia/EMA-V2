import 'package:ema_educacion_medica_avanzada/common/controllers.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/background_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class StartScreen extends StatelessWidget {
  final StartController controller = Get.put(StartController());

  StartScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final screenHeight = MediaQuery.sizeOf(context).height;

    final double logoHeight = screenHeight * 0.1;

    return BackgroundWidget(
      color: Colors.black,
      child: SafeArea(
        child: SizedBox(
          height: screenHeight,
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Image.asset(
                'assets/images/logotype_white.png',
                height: logoHeight,
              ),
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 32),
                child: Obx(() {
                  return LinearProgressIndicator(
                    value: controller.loading.value,
                    valueColor: AlwaysStoppedAnimation<Color>(AppStyles.tertiaryColor),
                    backgroundColor: AppStyles.grey220,
                    borderRadius: BorderRadius.circular(8),
                    semanticsLabel: 'Progreso del cuestionario',
                    semanticsValue: controller.loading.value.toString(),
                    minHeight: 6,
                  );
                }),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
