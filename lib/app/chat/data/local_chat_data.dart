// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/chat/interfaces/chat_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';

class LocalChatData extends ILocalData<ChatModel> implements IChatLocalData {
  static final String _tableName = 'chats_v1';
  @override String get tableName => _tableName;
  @override String get singular => 'el chat';
  @override String get plural => 'los chats';
  
  @override
  ChatModel fromApi(Map<String, dynamic> map) {
    return ChatModel.fromMap(map);
  }

  @override
  ChatModel fromMap(Map<String, dynamic> map) {
    return ChatModel.fromMap(map);
  }

  @override
  Map<String, dynamic> toMap(item) {
    // Todo: agregar try catch
    final chat = item as ChatModel;
    return chat.toMap();
  }

  static String sqlInstructionCreateTable() {
    return '''
      CREATE TABLE $_tableName (
        uid TEXT PRIMARY KEY,
        threadId TEXT,
        userId INTEGER NOT NULL,
        shortTitle TEXT NOT NULL,
        createdAt INTEGER NOT NULL,
        updatedAt INTEGER NOT NULL
      );
    ''';
  }
}
