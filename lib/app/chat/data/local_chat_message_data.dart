// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/chat/interfaces/chat_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';

class LocalChatMessageData extends ILocalData<ChatMessageModel>
    implements IChatMessageLocalData {
  static final String _tableName = 'chat_messages_v1';
  @override
  String get tableName => _tableName;
  @override
  String get singular => 'el mensaje';
  @override
  String get plural => 'los mensajes';

  @override
  ChatMessageModel fromApi(Map<String, dynamic> map) {
    return ChatMessageModel.fromApi(map);
  }

  @override
  ChatMessageModel fromMap(Map<String, dynamic> map) {
    return ChatMessageModel.fromMap(map);
  }

  @override
  Map<String, dynamic> toMap(item) {
    final chatMessage = item as ChatMessageModel;
    return chatMessage.toMap();
  }

  static String sqlInstructionCreateTable() {
    return '''
      CREATE TABLE $_tableName (
        uid TEXT PRIMARY KEY,
        chatId TEXT NOT NULL,
        text TEXT NOT NULL,
        aiMessage INTEGER NOT NULL,
        createdAt INTEGER NOT NULL,
        updatedAt INTEGER NOT NULL,
        attach TEXT,
        imageAttach TEXT
      );
    ''';
  }
}
