import 'package:ema_educacion_medica_avanzada/app/chat/chat.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/core/logger/logger.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


// Se está omitiendo para hacer pruebas
// el scrollController no funciona
class ChatList extends StatefulWidget {

  const ChatList({
    super.key,
  });

  @override
  State<ChatList> createState() => _ChatListState();

}

class _ChatListState extends State<ChatList> {
  final controller = Get.find<ChatController>();
  final scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    controller.setScrollController(scrollController);
  }

  @override
  void dispose() {
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    Logger.error('ChatList.build()');
    controller.setScrollController(scrollController);

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
