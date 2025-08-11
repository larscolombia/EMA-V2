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

    final response = await _dio.post(
      endpoint,
      data: {'prompt': prompt},
    );

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
  }) async {
    const endpoint = '/asistente/message';

    try {
      final data = {
        'thread_id': threadId,
        'prompt': prompt,
      };

      _cancelTokens[threadId] ??= CancelToken();
      final token = cancelToken ?? _cancelTokens[threadId]!;

      final response = await _dio.post(
        endpoint,
        data: data,
        cancelToken: token,
      );

      _cancelTokens.remove(threadId);

      if (response.statusCode == 200) {
        final text = response.data['text'] as String;
        return ChatMessageModel.ai(chatId: threadId, text: text);
      } else {
        throw Exception('Error al enviar mensaje');
      }
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

      final response = await _dio.post(
        endpoint,
        data: formData,
        cancelToken: token,
        onSendProgress: onSendProgress,
        options: dio.Options(
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        ),
      );

      // print('Response: ${response.statusCode}');
      // print('Response: ${response.data}');

      _cancelTokens.remove(threadId);
      if (response.statusCode == 200) {
        final text = response.data['text'] as String? ??
            "PDF cargado exitosamente. Puedes hacerme preguntas sobre su contenido ahora.";
        return ChatMessageModel.ai(chatId: threadId, text: text);
      } else if (response.statusCode == 500) {
        throw Exception('Error del servidor: ${response.data['message']}');
      } else {
        throw Exception('Error al cargar el PDF: ${response.statusCode}');
      }
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
