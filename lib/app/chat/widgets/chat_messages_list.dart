import 'package:ema_educacion_medica_avanzada/app/chat/chat.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

// Todo: ELIMINAR ESTE WIDGET Y USAR SOLO EL WIDGET DE LISTA

class ChatMessagesList extends GetView<ChatController> {
  const ChatMessagesList({
    super.key
  });

  Widget onLoading(bool loading) {
    return loading
      ? const Center(child: CircularProgressIndicator())
      : const SizedBox();
  }

  Widget onError(String? errorMessage) {
    return errorMessage != null && errorMessage.isNotEmpty
      ? Center(
          child: Text(errorMessage),
        )
      : const SizedBox();
  }

  @override
  Widget build(BuildContext context) {
    return Obx(() {

      List<Widget> children() {
        return controller.messages.map((message) {
          if (message.aiMessage) {
            return ChatMessageAi(message: message);
          } else {
            return ChatMessageUser(message: message);
          }
        }).toList();
      }

      return Container(
        padding: EdgeInsets.only(top: 16, left: 12, right: 12, bottom: 120),
        child: Column(
          mainAxisSize: MainAxisSize.max,
          mainAxisAlignment: MainAxisAlignment.end,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: children(),
        ),
      );
      // return ListView.builder(
      //   itemCount: controller.messages.length,
      //   shrinkWrap: true,
      //   padding: EdgeInsets.symmetric(horizontal: 12, vertical: 16),
      //   itemBuilder: (context, index) {
      //     final message = controller.messages[index];

      //     if (message.type == ChatMessageType.ai) {
      //       return ChatMessageAi(message: message);
      //     } else {
      //       return ChatMessageUser(message: message);
      //     }
      //   },
      // );
    });
  }
}
