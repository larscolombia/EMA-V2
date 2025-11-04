// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/actions/models/action_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_type.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/services/actions_service.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_message_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/interfaces/chat_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_start_response.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/image_attachment.dart';
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
      where: where,
      whereArgs: whereArgs,
    );

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
    ImageAttachment? image,
    void Function(String token)? onStream,
    String? focusDocId,
  }) async {
    // Persisting of both user and AI messages is handled by the controller to avoid duplicates
    // and to ensure that streamed content is not overwritten.

    if (image != null) {
      final apiMessage = await apiChatService.sendImageUpload(
        threadId: threadId,
        prompt: userMessage.text,
        image: image,
        onStream: onStream,
      );
      return ChatMessageModel.ai(
        chatId: userMessage.chatId,
        text: apiMessage.text,
      );
    }

    if (file != null) {
      final apiMessage = await apiChatService.sendPdfUpload(
        threadId: threadId,
        prompt: userMessage.text,
        file: file,
        onStream: onStream,
        focusDocId: focusDocId,
      );
      // Return the API response but DO NOT persist here.
      // The controller handles persistence after streaming completes.
      return ChatMessageModel.ai(
        chatId: userMessage.chatId,
        text: apiMessage.text,
      );
    }

    final apiMessage = await apiChatService.sendMessage(
      threadId: threadId,
      prompt: userMessage.text,
      onStream: onStream,
      focusDocId: focusDocId,
    );
    // Return the API response but DO NOT persist here.
    // The controller handles persistence after streaming completes.
    return ChatMessageModel.ai(
      chatId: userMessage.chatId,
      text: apiMessage.text,
    );
  }

  /// Delete a chat and all its local messages by chatId (uid).
  /// Also removes the associated Action record(s).
  /// CRÍTICO: También elimina thread y artefactos de OpenAI (vector stores de PDFs del usuario, files)
  /// pero PRESERVA el vector store compartido de libros médicos.
  Future<void> deleteChat(String chatId) async {
    // 1. Obtener threadId ANTES de borrar el chat localmente
    String? threadId;
    try {
      final chat = await chatsLocalData.getById('uid = ?', [chatId]);
      threadId = chat?.threadId;
    } catch (e) {
      print('⚠️ [deleteChat] Error obteniendo threadId: $e');
    }

    // 2. Borrar artefactos de OpenAI si existe threadId
    if (threadId != null && threadId.isNotEmpty) {
      try {
        print('🗑️ [deleteChat] Eliminando thread de OpenAI: $threadId');
        await apiChatService.deleteThread(threadId);
        print('✅ [deleteChat] Thread eliminado de OpenAI: $threadId');
      } catch (e) {
        // No fallar si no se puede borrar de OpenAI (best-effort)
        // El usuario puede estar offline o el thread ya no existir
        print('⚠️ [deleteChat] Error al eliminar thread de OpenAI: $e');
      }
    }

    // 3. Borrar datos locales (mensajes, chat, acciones)
    await chatMessagesLocalData.delete(
      where: 'chatId = ?',
      whereArgs: [chatId],
    );
    await chatsLocalData.delete(where: 'uid = ?', whereArgs: [chatId]);
    await actionsService.deleteActionsByItemId(ActionType.chat, chatId);
  }
}
