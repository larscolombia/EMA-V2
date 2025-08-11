/*import 'dart:math';

import 'package:dio/src/cancel_token.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/interfaces/chat_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';

class FakerApiChatService extends IApiChatData {
  // @override
  // Future<ChatModel> generateNewChat(ChatModel chat, [String? message, PdfAttachment? file]) async {
  //   await Future.delayed(Duration(seconds: 2));

  //   return ChatModel.generate(userId: chat.userId, message: chat.shortTitle);
  // }

  @override
  Future<List<ChatModel>> getChatsByUserId({required String userId}) async {
    await Future.delayed(Duration(seconds: 2));

    List<ChatModel> chats = [
      ChatModel.empty(5).copyWith(shortTitle: 'Paciente con algo'),
      ChatModel.empty(5).copyWith(shortTitle: 'Paciente sin algo'),
      ChatModel.empty(5).copyWith(shortTitle: 'Paciente indeciso'),
    ];
    return chats;
  }

  @override
  Future<ChatModel?> getChatById(String id) async {
    return ChatModel.empty(5).copyWith(shortTitle: 'Paciente con algo');
  }

  @override
  Future<List<ChatMessageModel>> getMessagesById({required String id}) async {
    await Future.delayed(Duration(seconds: 2));

    List<ChatMessageModel> chatMessages = [
      ChatMessageModel.user(
        chatId: id,
        text:
            'Lorem ipsum dolor sit amet consectetur adipiscing elit, dapibus non in inceptos fusce luctus egestas dictum, sed id quis rhoncus mus taciti. Malesuada blandit rhoncus fames viverra torquent vel praesent, ultrices tempus parturient ac facilisi pellentesque, erat pulvinar condimentum bibendum platea sociis.',
      ),
      ChatMessageModel.ai(
        chatId: id,
        text:
            'Cubilia ante netus vel quis eleifend arcu inceptos lectus semper, facilisi cursus parturient metus purus dignissim himenaeos nibh tortor leo, nam duis vehicula in ac cras sociis sapien.',
      ),
      ChatMessageModel.user(
        chatId: id,
        text:
            'Erat aptent nec consequat pretium sapien pulvinar quisque suscipit, himenaeos iaculis lectus auctor magnis habitant nam volutpat accumsan.',
      ),
      ChatMessageModel.ai(
        chatId: id,
        text:
            'Felis cum augue ut torquent mollis dui non vivamus platea sollicitudin donec, taciti habitant nisl eleifend lobortis dignissim turpis scelerisque metus sagittis, facilisi quis duis vitae eros pulvinar nam magnis sodales cras.',
      ),
      ChatMessageModel.user(
        chatId: id,
        text:
            'Porta habitant ut suscipit cubilia mattis senectus, justo dapibus in euismod vel venenatis, vehicula inceptos tincidunt metus dictum. Interdum quam.',
      ),
      ChatMessageModel.ai(
        chatId: id,
        text:
            'Consequat facilisi ornare quis himenaeos ultricies sem vestibulum, habitasse lectus aptent faucibus et. Dui augue libero purus auctor per mollis senectus enim, fermentum eget habitasse in conubia etiam vivamus porta nec, vehicula interdum ullamcorper pharetra neque posuere fames.',
      ),
    ];
    return chatMessages;
  }

  @override
  Future<ChatMessageModel> sendMessage(ChatMessageModel userMessage, CancelToken
      [PdfAttachment? file]) async {
    await Future.delayed(Duration(seconds: 1));
    return _emulateAiMessage(chatId: userMessage.chatId);
  }

  ChatMessageModel _emulateAiMessage({required String chatId}) {
    List<String> aiMessages = [
      'Lorem ipsum dolor sit amet consectetur adipiscing elit, dapibus non in inceptos fusce luctus egestas dictum, sed id quis rhoncus mus taciti. Malesuada blandit rhoncus fames viverra torquent vel praesent, ultrices tempus parturient ac facilisi pellentesque, erat pulvinar condimentum bibendum platea sociis.',
      'Cubilia ante netus vel quis eleifend arcu inceptos lectus semper, facilisi cursus parturient metus purus dignissim himenaeos nibh tortor leo, nam duis vehicula in ac cras sociis sapien.',
      'Erat aptent nec consequat pretium sapien pulvinar quisque suscipit, himenaeos iaculis lectus auctor magnis habitant nam volutpat accumsan.',
      'Felis cum augue ut torquent mollis dui non vivamus platea sollicitudin donec, taciti habitant nisl eleifend lobortis dignissim turpis scelerisque metus sagittis, facilisi quis duis vitae eros pulvinar nam magnis sodales cras.',
    ];

    final aiMessage = aiMessages[Random().nextInt(aiMessages.length)];

    return ChatMessageModel.ai(
      chatId: chatId,
      text: aiMessage,
    );
  }

  @override
  Future<ChatMessageModel> sendPdfUpload(
      ChatMessageModel userMessage, PdfAttachment file,
      {CancelToken? cancelToken, Function(int p1, int p2)? onSendProgress}) {
    // TODO: implement sendPdfUpload
    throw UnimplementedError();
  }
}*/
