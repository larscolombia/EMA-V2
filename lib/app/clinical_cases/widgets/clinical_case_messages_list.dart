import 'package:ema_educacion_medica_avanzada/app/chat/chat.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ClinicalCaseMessageList extends StatefulWidget {

  const ClinicalCaseMessageList({
    super.key,
  });

  @override
  State<ClinicalCaseMessageList> createState() => _ClinicalCaseMessageListState();

}

class _ClinicalCaseMessageListState extends State<ClinicalCaseMessageList> {
  final controller = Get.find<ClinicalCaseController>();
  final scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    controller.setScrollController(scrollController);
  }

  @override
  void dispose() {
    controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Obx(() {
      return ListView.builder(
        itemCount: controller.messages.length,
        controller: scrollController,
        padding: EdgeInsets.only(left: 12, right: 12, top: 16),

        itemBuilder: (context, index) {
          final message = controller.messages[index];
          // Ocultar el prompt interno usado para generar la evaluación final analítica
          // cuando el caso ya está finalizado (o se generó la evaluación). Evitamos
          // mostrar texto largo y técnico que no aporta al usuario.
          if (!message.aiMessage && controller.isComplete.value) {
            final lower = message.text.trimLeft().toLowerCase();
            if (lower.startsWith('genera una evaluación final detallada') || lower.startsWith('[[hidden_eval_prompt]]')) {
              return const SizedBox.shrink();
            }
          }
          return message.aiMessage
            ? ChatMessageAi(message: message)
            : ChatMessageUser(message: message);
        },
      );
    });
  }
}
