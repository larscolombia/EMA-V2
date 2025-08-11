// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/actions/models/action_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/services/actions_service.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_message_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/interfaces/chat_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_start_response.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:get/get.dart';

class ChatHelper {
  static String shortTitle(String? message, PdfAttachment? file) {
    if (message != null && message.isNotEmpty) {
      return message.split(' ').take(5).join(' ');
    } else if (file != null && file.fileName.isNotEmpty) {
      return 'Chat sobre el archivo ${file.fileName}';
    }
    return 'Chat...';
  }
}

class ChatsService extends GetxService {
  final actionsService = Get.find<ActionsService>();
  final apiChatService = Get.find<IApiChatData>();
  final userService = Get.find<UserService>();
  final chatsLocalData = Get.find<LocalChatData>();
  final chatMessagesLocalData = Get.find<LocalChatMessageData>();

  Future<ChatStartResponse> startChat(String prompt) async {
    final response = await apiChatService.startChat(prompt);
    return response;
  }

  Future<ChatModel> generateNewChat(
    ChatModel currentChat,
    String? userText,
    PdfAttachment? file,
    String threadId,
  ) async {
    final userId = userService.currentUser.value.id;
    final shortTitle = ChatHelper.shortTitle(userText, file);

    final newChat = currentChat.copyWith(
      userId: userId,
      shortTitle: shortTitle,
      threadId: threadId,
    );
    await chatsLocalData.insertOne(newChat);

    final action = ActionModel.chat(userId, newChat.uid, newChat.shortTitle);
    actionsService.insertAction(action);

    return newChat;
  }

  Future<ChatModel?> getChatById(String id) async {
    final where = 'uid = ?';
    final whereArgs = [id];

    final localChat = await chatsLocalData.getById(where, whereArgs);

    if (localChat != null) {
      return localChat;
    }

    final remoteChat = await apiChatService.getChatById(id);

    if (remoteChat == null) {
      return null;
    } else {
      await chatsLocalData.insertOne(remoteChat);
    }

    return remoteChat;
  }

  Future<List<ChatMessageModel>> getMessagesById(String id) async {
    final where = 'chatId = ?';
    final whereArgs = [id];

    final localMessages = await chatMessagesLocalData.getItems(
        where: where, whereArgs: whereArgs);

    if (localMessages.isNotEmpty) {
      return localMessages;
    }

    final remoteMessages = await apiChatService.getMessagesById(id: id);

    if (remoteMessages.isNotEmpty) {
      await chatMessagesLocalData.insertMany(remoteMessages);
      return remoteMessages;
    } else {
      return [];
    }
  }

  Future<ChatMessageModel> sendMessage({
    required String threadId,
    required ChatMessageModel userMessage,
    PdfAttachment? file,
  }) async {
    chatMessagesLocalData.insertOne(userMessage);

    if (file != null) {
      final apiMessage = await apiChatService.sendPdfUpload(
        threadId: threadId,
        prompt: userMessage.text,
        file: file,
      );
      final aiPdfMessage = ChatMessageModel.ai(
        chatId: userMessage.chatId,
        text: apiMessage.text,
      );
      chatMessagesLocalData.insertOne(aiPdfMessage);
      return aiPdfMessage;
    }

    final apiMessage = await apiChatService.sendMessage(
      threadId: threadId,
      prompt: userMessage.text,
    );
    final aiMessage = ChatMessageModel.ai(
      chatId: userMessage.chatId,
      text: apiMessage.text,
    );
    chatMessagesLocalData.insertOne(aiMessage);

    return aiMessage;
  }
}
