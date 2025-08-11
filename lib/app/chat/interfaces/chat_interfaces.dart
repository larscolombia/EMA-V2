import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_start_response.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
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
  });

  Future<ChatMessageModel> sendPdfUpload({
    required String threadId,
    required String prompt,
    required PdfAttachment file,
    CancelToken? cancelToken,
    Function(int, int)? onSendProgress,
  });
}
