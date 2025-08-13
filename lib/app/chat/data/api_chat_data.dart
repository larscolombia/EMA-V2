import 'dart:convert';
import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart' as dio;
import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/interfaces/chat_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_start_response.dart';
import 'package:ema_educacion_medica_avanzada/core/api/api_service.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:get/get.dart';

class ApiChatData implements IApiChatData {
  final _dio = Get.find<ApiService>().dio;
  final userService = Get.find<UserService>();

  final Map<String, CancelToken> _cancelTokens = {};

  @override
  Future<ChatStartResponse> startChat(String prompt) async {
    const endpoint = '/asistente/start';

    final response = await _dio.post(endpoint, data: {'prompt': prompt});

    if (response.statusCode == 200) {
      return ChatStartResponse(
        threadId: response.data['thread_id'] as String,
        text: response.data['text'] as String,
      );
    } else {
      throw Exception('Error al iniciar chat');
    }
  }

  @override
  Future<ChatModel?> getChatById(String id) async {
    // Todo: obtener el chat del servidor
    return null;
  }

  @override
  Future<List<ChatModel>> getChatsByUserId({required String userId}) async {
    List<ChatModel> chats = [];
    return chats;
  }

  @override
  Future<List<ChatMessageModel>> getMessagesById({required String id}) async {
    List<ChatMessageModel> chatMessages = [];
    return chatMessages;
  }

  @override
  Future<ChatMessageModel> sendMessage({
    required String threadId,
    required String prompt,
    CancelToken? cancelToken,
    void Function(String token)? onStream,
  }) async {
    const endpoint = '/asistente/message';

    try {
      final data = {'thread_id': threadId, 'prompt': prompt};

      _cancelTokens[threadId] ??= CancelToken();
      final token = cancelToken ?? _cancelTokens[threadId]!;

      final response = await _dio.post<dio.ResponseBody>(
        endpoint,
        data: data,
        cancelToken: token,
        options: Options(
          responseType: ResponseType.stream,
          headers: {'Accept': 'text/event-stream'},
        ),
      );

      _cancelTokens.remove(threadId);

      final body = response.data;
      if (body == null) {
        throw Exception('Respuesta de streaming vacía');
      }
      // body.stream is a Stream<List<int>>; decode as UTF8
      final stream = utf8.decoder.bind(body.stream);
      final buffer = StringBuffer();

      await for (final chunk in stream) {
        for (final line in const LineSplitter().convert(chunk)) {
          if (line.startsWith('data:')) {
            // Preserve original token spacing. Our SSE writer sends "data: <msg>";
            // remove only the single space after the colon to restore the exact token from OpenAI.
            var content = line.substring(5); // " <msg>" or " [DONE]"
            if (content.startsWith(' ')) content = content.substring(1);
            if (content == '[DONE]') {
              break;
            }
            buffer.write(content);
            onStream?.call(content);
          }
        }
      }

      return ChatMessageModel.ai(chatId: threadId, text: buffer.toString());
    } on DioException catch (e) {
      if (CancelToken.isCancel(e)) {
        rethrow; // Let the controller handle cancellation
      }
      throw Exception('Error en la comunicación: ${e.message}');
    } catch (e) {
      throw Exception('Error inesperado: $e');
    }
  }

  @override
  Future<ChatMessageModel> sendPdfUpload({
    required String threadId,
    required String prompt,
    required PdfAttachment file,
    CancelToken? cancelToken,
    Function(int, int)? onSendProgress,
    void Function(String token)? onStream,
  }) async {
    const endpoint = '/asistente/message';
    try {
      // Verificamos que el archivo existe
      final pdfFile = File(file.filePath);
      if (!await pdfFile.exists()) {
        throw Exception('El archivo PDF no existe en la ruta especificada');
      }

      // Preparamos el FormData para la carga del archivo
      final formData = dio.FormData.fromMap({
        'file': await dio.MultipartFile.fromFile(
          file.filePath,
          filename: file.fileName,
        ),
        'prompt': prompt,
        'thread_id': threadId,
      });

      // formData.fields.forEach((field) {
      //   print('Field: ${field.key}, Value: ${field.value}');
      // });
      // formData.files.forEach((file) {
      //   print('File: ${file.key}, Filename: ${file.value.filename}');
      // });

      _cancelTokens[threadId] ??= CancelToken();
      final token = cancelToken ?? _cancelTokens[threadId]!;

      final response = await _dio.post<dio.ResponseBody>(
        endpoint,
        data: formData,
        cancelToken: token,
        onSendProgress: onSendProgress,
        options: dio.Options(
          headers: {
            'Content-Type': 'multipart/form-data',
            'Accept': 'text/event-stream',
          },
          responseType: ResponseType.stream,
        ),
      );

      _cancelTokens.remove(threadId);
      final body = response.data;
      if (body == null) {
        throw Exception('Respuesta de streaming vacía');
      }
      final stream = utf8.decoder.bind(body.stream);
      final buffer = StringBuffer();
      await for (final chunk in stream) {
        for (final line in const LineSplitter().convert(chunk)) {
          if (line.startsWith('data:')) {
            var content = line.substring(5);
            if (content.startsWith(' ')) content = content.substring(1);
            if (content == '[DONE]') {
              break;
            }
            buffer.write(content);
            onStream?.call(content);
          }
        }
      }
      return ChatMessageModel.ai(chatId: threadId, text: buffer.toString());
    } on DioException catch (e) {
      if (CancelToken.isCancel(e)) {
        rethrow; // Let the controller handle cancellation
      }
      throw Exception('Error en la comunicación: ${e.message}');
    } catch (e) {
      throw Exception('Error inesperado: $e');
    }
  }
}
