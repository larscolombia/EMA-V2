import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_start_response.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/image_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';

abstract class IChatLocalData implements ILocalData<ChatModel> {}

abstract class IChatMessageLocalData implements ILocalData<ChatMessageModel> {}

abstract class IApiChatData {
  Future<ChatStartResponse> startChat(String prompt);

  Future<ChatModel?> getChatById(String id);

  Future<List<ChatModel>> getChatsByUserId({required String userId});

  Future<List<ChatMessageModel>> getMessagesById({required String id});

  Future<ChatMessageModel> sendMessage({
    required String threadId,
    required String prompt,
    CancelToken? cancelToken,
    void Function(String token)? onStream,
    String? focusDocId,
  });

  Future<ChatMessageModel> sendPdfUpload({
    required String threadId,
    required String prompt,
    required PdfAttachment file,
    CancelToken? cancelToken,
    Function(int, int)? onSendProgress,
    void Function(String token)? onStream,
    String? focusDocId,
  });

  Future<ChatMessageModel> sendImageUpload({
    required String threadId,
    required String prompt,
    required ImageAttachment image,
    CancelToken? cancelToken,
    Function(int, int)? onSendProgress,
    void Function(String token)? onStream,
  });

  /// Elimina thread y artefactos asociados de OpenAI
  /// Solo borra vector stores creados por el usuario (PDFs), NO el vector store de libros compartido
  Future<void> deleteThread(String threadId);
}
