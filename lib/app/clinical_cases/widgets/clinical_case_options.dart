import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/life_stage.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/sex_and_status.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/outline_ai_button.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class ClinicalCaseOptions extends StatelessWidget {
  final clinicalCaseController = Get.find<ClinicalCaseController>();
  // final keyboardService = Get.find<KeyboardService>();
  // final uiObserverService = Get.find<UiObserverService>();
  // final chatController = Get.find<ChatController>();

  final Rx<LifeStage> lifeStage = LifeStage.adulto.obs;
  final Rx<SexAndStatus> sexAndStatus = SexAndStatus.woman.obs;

  final ClinicalCaseType type;

  ClinicalCaseOptions({super.key, required this.type});

  void updateLifeStage(double value) {
    final newLifeStage = LifeStage.fromValue(value);
    lifeStage.value = newLifeStage;
    _syncSexAndStatusOptions(newLifeStage);
  }

  void updateSexAndStatus(double value) {
    final newSexAndStatus = SexAndStatus.fromValue(value);
    sexAndStatus.value = newSexAndStatus;
    _syncLifeStageOptions(newSexAndStatus);
  }

  void _syncSexAndStatusOptions(LifeStage currentLifeStage) {
    if (!currentLifeStage.maybePregnant && sexAndStatus.value.isPregnant) {
      // Si la etapa de vida no permite embarazo y SexAndStatus es una opción de embarazo,
      // forzamos a una opción no embarazada (ej. 'woman').
      sexAndStatus.value = SexAndStatus.woman;
    }
  }

  void _syncLifeStageOptions(SexAndStatus currentSexAndStatus) {
    if (currentSexAndStatus.isPregnant && !lifeStage.value.maybePregnant) {
      // Si SexAndStatus no permite embarazo y LifeStage es una opción de embarazo,
      // forzamos a una opción no embarazada (ej. 'woman').
      lifeStage.value = LifeStage.joven;
    }
  }

  void _generateClinicalCase() {
    print('🔘 [ClinicalCaseOptions] Generate case button pressed');
    clinicalCaseController.generateCase(
      type: type,
      lifeStage: lifeStage.value,
      sexAndStatus: sexAndStatus.value,
    );
  }

  @override
  Widget build(BuildContext context) {
    Widget buildLifeStageSlider() {
      return Row(
        children: [
          Expanded(
            flex: 2,
            child: Text(
              '${lifeStage.value.name}\n${lifeStage.value.description}',
            ),
          ),
          Expanded(
            flex: 3,
            child: Slider(
              value: lifeStage.value.value,
              min: LifeStage.prenatal.value,
              max: LifeStage.anciano.value,
              divisions:
                  LifeStage.values.length -
                  1, // Número de divisiones entre cada valor
              onChanged: (value) {
                updateLifeStage(value);
              },
            ),
          ),
        ],
      );
    }

    Widget buildSexAndStatusSlider() {
      return Row(
        children: [
          Expanded(
            flex: 2,
            child: Text(sexAndStatus.value.description, softWrap: false),
          ),
          Expanded(
            flex: 3,
            child: Slider(
              value: sexAndStatus.value.value,
              min: SexAndStatus.man.value,
              max: SexAndStatus.womanPregnantLateFetalPhase.value,
              divisions: SexAndStatus.values.length - 1,
              onChanged: (value) {
                updateSexAndStatus(value);
              },
            ),
          ),
        ],
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('PACIENTE', style: AppStyles.detailLabel),
        SizedBox(height: 8),
        Obx(() => buildSexAndStatusSlider()),
        Obx(() => buildLifeStageSlider()),
        SizedBox(height: 8),
        Obx(
          () => OutlineAiButton(
            text: 'Dame un caso',
            onPressed: _generateClinicalCase,
            enabled: !clinicalCaseController.isTyping.value,
          ),
        ),
      ],
    );
  }
}
