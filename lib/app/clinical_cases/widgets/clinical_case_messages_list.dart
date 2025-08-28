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
          return message.aiMessage
            ? ChatMessageAi(message: message)
            : ChatMessageUser(message: message);
        },
      );
    });
  }
}
