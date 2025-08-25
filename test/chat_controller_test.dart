import 'package:flutter_test/flutter_test.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/services/chats_service.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/attachment_service.dart';
import 'package:ema_educacion_medica_avanzada/core/ui/ui_observer_service.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/controllers/profile_controller.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_model.dart';

// Fake ChatsService to simulate responses
class FakeChatsService extends ChatsService {
  @override
  Future<ChatMessageModel> sendMessage({
    required String threadId,
    required ChatMessageModel userMessage,
    PdfAttachment? file,
    void Function(String token)? onStream,
  }) async {
    // Simulate streaming by sending two tokens
    final text =
        file != null
            ? 'No encontré información en el archivo adjunto.'
            : 'Sí, aquí estoy! Si tienes alguna pregunta...';
    if (onStream != null) {
      // simulate token by token
      for (var token in text.split(' ')) {
        onStream('$token ');
      }
    }
    return ChatMessageModel.ai(chatId: userMessage.chatId, text: text);
  }
}

void main() {
  late ChatController controller;

  setUp(() {
    Get.reset();
    // Provide fake dependencies
    // ChatsService with streaming behavior
    Get.put<ChatsService>(FakeChatsService());
    // AttachmentService stub
  Get.put<AttachmentService>(AttachmentService(validateFile: (file) async {}));
    // UI observer stub
  Get.put<UiObserverService>(UiObserverService(isKeyboardVisible: Stream<bool>.value(false)));
    // ProfileController stub: unlimited quotas
  Get.put<ProfileController>(ProfileController(initialProfile: Profile(id: 1, authToken: 'token')));
    // UserService stub
  Get.put<UserService>(UserService(currentUser: Rx(UserModel(id: 1, authToken: 'token'))));
    // Instantiate controller
    controller = ChatController();
    Get.put(controller);
    controller.threadId = 'thread1';
    // Seed initial message so not treated as new chat
    controller.messages.add(
      ChatMessageModel.user(chatId: 'thread1', text: 'inicio'),
    );
  });

  test('Should handle PDF conversation flow correctly', () async {
    final pdf = PdfAttachment(
      uid: 'p1',
      fileName: 'test.pdf',
      filePath: 'path',
      mimeType: 'application/pdf',
      fileSize: 10,
    );
    // Attach PDF and send first message
    controller.attachPdf(pdf);
    await controller.sendMessage('Que dice este doc');

    expect(controller.messages.length, 2);
    expect(controller.messages[0].aiMessage, false);
    expect(controller.messages[1].aiMessage, true);
    expect(
      controller.messages[1].text,
      'No encontré información en el archivo adjunto.',
    );

    // Send follow-up
    await controller.sendMessage('No dice nada?');
    expect(controller.messages.length, 4);
    expect(controller.messages[2].aiMessage, false);
    expect(controller.messages[3].aiMessage, true);
    expect(controller.messages[3].text.contains('Sí, aquí estoy'), true);
  });
}
