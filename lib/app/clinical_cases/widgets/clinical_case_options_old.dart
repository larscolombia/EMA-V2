// import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
// import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_type.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ClinicalCaseOptions extends StatelessWidget {
  // final keyboardService = Get.find<KeyboardService>();
  final clinicalCaseController = Get.find<ClinicalCaseController>();
  // final uiObserverService = Get.find<UiObserverService>();
  // final chatController = Get.find<ChatController>();

  final RxString clinicalCaseDescription = RxString('');

  final ClinicalCaseType type;

  ClinicalCaseOptions({
    super.key,
    required this.type,
  });

  void _generateClinicalCase() {
    if (type == ClinicalCaseType.analytical) {
      // Esto era solo una prueba
      // chatController.iniciarClinicalCaseAnalytic('Generar un caso clínico completo para analizarlo.');
    }
    Get.toNamed(Routes.home.name);
  }

  final _outlineEnableBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(16),
    borderSide: BorderSide(
      color: Colors.transparent,
    ),
  );

  final _outlineFocusBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(16),
    borderSide: BorderSide(
      color: AppStyles.primary900,
    ),
  );

  @override
  Widget build(BuildContext context) {
    final textController = TextEditingController();
    final focusNode = FocusNode();

    final buttons = Padding(
      padding: const EdgeInsets.only(right: 6),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            onPressed: () {
              textController.clear();
              clinicalCaseDescription.value = '';
            },
            padding: EdgeInsets.all(8),
            icon: AppIcons.closeSquare(
              height: 24,
              width: 24,
              color: AppStyles.tertiaryColor,
            ),
          ),
        ],
      ),
    );

    final inputDecoration = InputDecoration(
      label: Text('Describa el caso que desea analizar'),
      enabledBorder: _outlineEnableBorder,
      focusedBorder: _outlineFocusBorder,
      contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 0),
      floatingLabelBehavior: FloatingLabelBehavior.never,
      suffixIcon: buttons,
      filled: true,
    );

    final textFormField = TextFormField(
      autocorrect: false,
      focusNode: focusNode,
      controller: textController,

      decoration: inputDecoration,
      keyboardType: TextInputType.text,
      maxLines: 1,

      onFieldSubmitted: (value) {
        clinicalCaseDescription.value = value;
        focusNode.unfocus();
      },

      onChanged: (value) {
        clinicalCaseDescription.value = value;
      },

      onEditingComplete: () {
        final value = textController.text;
        clinicalCaseDescription.value = value;
      },

      onTapOutside: (event) {
        focusNode.unfocus();
      },
    );

    return Column(
      children: [
        textFormField,

        const SizedBox(height: 12),

        Obx(() {
          final bool enabled = clinicalCaseDescription.value.isNotEmpty;

          return OutlineAiButton(
            text: 'Dame un caso',
            onPressed: _generateClinicalCase,
            enabled: enabled,
          );
        }),
      ],
    );
  }
}
