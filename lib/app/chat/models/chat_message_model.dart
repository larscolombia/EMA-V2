// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:flutter/widgets.dart';
import 'package:uuid/uuid.dart';

class ChatMessageModel {
  final String uid;
  final String chatId;
  final String text;
  final bool aiMessage;
  final DateTime createdAt;
  final DateTime updatedAt;
  final PdfAttachment? attach;
  final Widget? widget; // Widget personalizado para renderizar el mensaje

  ChatMessageModel._({
    required this.uid,
    required this.chatId,
    required this.text,
    required this.aiMessage,
    required this.createdAt,
    required this.updatedAt,
    this.attach,
    this.widget,
  });

  factory ChatMessageModel.ai({
    required String chatId,
    required String text,
    Widget? widget,
  }) {
    return ChatMessageModel._(
      uid: const Uuid().v4(),
      chatId: chatId,
      text: text,
      aiMessage: true,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      attach: null,
      widget: widget,
    );
  }

  // Todo: pendiente mapear este constructor a partir del endpoint recuperar hilo
  factory ChatMessageModel.fromApi(Map<String, dynamic> map) {
    return ChatMessageModel._(
      uid: Uuid().v4(),
      chatId: map['chatId']
          as String, // Todo: verificar si entrega chatId o cuestionario_id
      text: map['text'] as String,
      aiMessage: map['role'] == 'assistant',
      createdAt: DateTime.fromMillisecondsSinceEpoch(map['createdAt'] as int),
      updatedAt: DateTime.fromMillisecondsSinceEpoch(map['updatedAt'] as int),
      attach: null,
    );
  }

  factory ChatMessageModel.temporal(String text, bool aiMessage) {
    return ChatMessageModel._(
      uid: "__temporal__",
      chatId: "",
      text: text,
      aiMessage: aiMessage,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      attach: null,
    );
  }

  factory ChatMessageModel.user({
    required String chatId,
    required String text,
    PdfAttachment? attach,
  }) {
    return ChatMessageModel._(
      uid: const Uuid().v4(),
      chatId: chatId,
      text: text,
      aiMessage: false,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      attach: attach,
    );
  }

  factory ChatMessageModel.fromMap(Map<String, dynamic> map) {
    return ChatMessageModel._(
      uid: map['uid'] as String,
      chatId: map['chatId'] as String,
      text: map['text'] as String,
      aiMessage: map['aiMessage'] == 1,
      createdAt: DateTime.fromMillisecondsSinceEpoch(map['createdAt'] as int),
      updatedAt: DateTime.fromMillisecondsSinceEpoch(map['updatedAt'] as int),
      attach: map['attach'] != null
          ? PdfAttachment.fromJson(map['attach'] as String)
          : null, // Todo: desde json string
    );
  }

  factory ChatMessageModel.fromJson(String source) =>
      ChatMessageModel.fromMap(json.decode(source) as Map<String, dynamic>);

  Map<String, dynamic> toBody(int userId) {
    return {
      "user_id": userId,
      "id_conversation": chatId,
      'prompt': text,
      if (attach != null)
        'attachment': {
          'fileName': attach!.fileName,
          'mimeType': attach!.mimeType,
          'filePath': attach!.filePath,
        },
    };
  }

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'uid': uid,
      'chatId': chatId,
      'text': text,
      'aiMessage': aiMessage ? 1 : 0,
      'createdAt': createdAt.millisecondsSinceEpoch,
      'updatedAt': updatedAt.millisecondsSinceEpoch,
      'attach': attach?.toJson(), // Todo: almacenar como json string
    };
  }

  String toJson() => json.encode(toMap());

  @override
  String toString() {
    return 'uid: $uid,\ntext: $text, aiMessage: $aiMessage';
  }
}
