import 'dart:convert';
import 'dart:async';

import 'package:dio/dio.dart' as dio;
import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_generate_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/controllers/subscription_controller.dart';
import 'package:ema_educacion_medica_avanzada/core/api/api_service.dart';
import 'package:ema_educacion_medica_avanzada/core/logger/logger.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:get/get.dart';

class ApiClinicalCaseData {
  final Dio _dio = Get.find<ApiService>().dio;
  final Map<String, CancelToken> _cancelTokens = {};

  Future<ClinicalCaseGenerateData> generateCase(
    ClinicalCaseModel clinicalCase,
  ) async {
    try {
      final body = clinicalCase.toRequestBody();

      // Opción A: usar flujo nuevo con contrato estricto para casos interactivos
      final url =
          clinicalCase.type == ClinicalCaseType.interactive
              ? '/casos-interactivos/iniciar'
              : '/caso-clinico';

      final response = await _dio.post(url, data: body);

      final generateClinicalCase = ClinicalCaseModel.fromApi(response.data);

      final completeClinicalCase = generateClinicalCase.copyWith(
        type: clinicalCase.type,
        uid: clinicalCase.uid,
        userId: clinicalCase.userId,
      );

      QuestionResponseModel? firstQuestion;
      if (clinicalCase.type == ClinicalCaseType.interactive) {
        // Nuevo contrato: data { feedback, next { hallazgos, pregunta } }
        final data =
            (response.data['data'] ?? const <String, dynamic>{})
                as Map<String, dynamic>;
        final next =
            (data['next'] ?? const <String, dynamic>{}) as Map<String, dynamic>;
        final pregunta = next['pregunta'] as Map<String, dynamic>?;
        final feedback = data['feedback'] as String? ?? '';

        if (pregunta != null) {
          final questionMap = {
            'id': 0,
            'question': pregunta['texto'] ?? '',
            'answer': '',
            'type': (pregunta['tipo'] ?? 'single_choice').toString().replaceAll(
              '-',
              '_',
            ),
            'options': pregunta['opciones'] ?? [],
          };
          firstQuestion = QuestionResponseModel.fromClinicalCaseApi(
            quizId: clinicalCase.uid,
            feedback: feedback,
            questionMap: questionMap,
          );
        }
      }

      try {
        final storage = const FlutterSecureStorage();
        final key =
            clinicalCase.type == ClinicalCaseType.interactive
                ? 'interactive_strict_thread_id'
                : 'interactive_case_thread_id';
        await storage.write(key: key, value: generateClinicalCase.threadId);
      } catch (_) {}

      return ClinicalCaseGenerateData(
        clinicalCase: completeClinicalCase,
        question: firstQuestion,
      );
    } catch (e) {
      final msg = e.toString();
      Logger.error(msg);
      // Detección básica de cuota agotada (mensaje backend 403)
      if (msg.contains('clinical cases quota exceeded') ||
          msg.contains('quota') ||
          msg.contains('403')) {
        // Intentar notificar y redirigir a planes
        try {
          final subController =
              Get.isRegistered<SubscriptionController>()
                  ? Get.find<SubscriptionController>()
                  : null;
          subController?.handleQuotaExceeded();
        } catch (_) {}
      }
      throw Exception('No fue posible crear el caso clínico.');
    }
  }

  Future<QuestionResponseModel> sendAnswerMessage(
    QuestionResponseModel questionWithAnswer,
  ) async {
    // Flujo nuevo: /casos-interactivos/mensaje
    final storage = const FlutterSecureStorage();
    final threadId = await storage.read(key: 'interactive_strict_thread_id');

    final Map<String, dynamic> body = {
      'thread_id': threadId,
      'mensaje': questionWithAnswer.message,
    };

    // Añadir answer_index si es single choice y la opción existe
    try {
      if (questionWithAnswer.type.name == 'single_choice' &&
          questionWithAnswer.userAnswer != null &&
          questionWithAnswer.userAnswer!.isNotEmpty) {
        final idx = questionWithAnswer.options.indexOf(
          questionWithAnswer.userAnswer!,
        );
        if (idx >= 0) {
          body['answer_index'] = idx; // índice 0-based int
        }
      }
    } catch (_) {}

    Logger.objectValue(
      'body_enviado_al_endpoint: /casos-interactivos/mensaje',
      body.toString(),
    );

    final response = await _dio.post('/casos-interactivos/mensaje', data: body);

    if (response.statusCode == 200) {
      final data = response.data['data'] ?? response.data;
      final feedback = data['feedback'] as String? ?? '';
      final next =
          (data['next'] ?? const <String, dynamic>{}) as Map<String, dynamic>;
      final pregunta =
          (next['pregunta'] ?? const <String, dynamic>{})
              as Map<String, dynamic>;
      final questionMap = {
        'id': 0,
        'question': pregunta['texto'] ?? '',
        'answer': '',
        'type': (pregunta['tipo'] ?? 'single_choice').toString().replaceAll(
          '-',
          '_',
        ),
        'options': pregunta['opciones'] ?? [],
      };

      final newThreadId = data['thread_id'];
      if (newThreadId is String && newThreadId.isNotEmpty) {
        try {
          await storage.write(
            key: 'interactive_strict_thread_id',
            value: newThreadId,
          );
        } catch (_) {}
      }

      return QuestionResponseModel.fromClinicalCaseApi(
        quizId: questionWithAnswer.quizId,
        feedback: feedback,
        questionMap: questionMap,
      );
    } else {
      throw Exception('Error al enviar mensaje');
    }
  }

  Future<ChatMessageModel> sendMessage(
    ChatMessageModel userMessage, {
    CancelToken? cancelToken,
    void Function(String token)? onStream,
  }) async {
    final storage = const FlutterSecureStorage();
    final threadId = await storage.read(key: 'interactive_case_thread_id');

    final body = {'thread_id': threadId, 'mensaje': userMessage.text};

    try {
      _cancelTokens[threadId ?? ''] ??= CancelToken();
      final token = cancelToken ?? _cancelTokens[threadId ?? '']!;

      final response = await _dio.post<dio.ResponseBody>(
        '/casos-clinicos/conversar',
        data: body,
        cancelToken: token,
        options: Options(
          responseType: ResponseType.stream,
          headers: {'Accept': 'text/event-stream'},
        ),
      );

      _cancelTokens.remove(threadId);

      final bodyStream = response.data;
      if (bodyStream == null) {
        throw Exception('Respuesta de streaming vacía');
      }

      final stream = utf8.decoder.bind(bodyStream.stream);
      final buffer = StringBuffer();

      await for (final chunk in stream) {
        for (final line in const LineSplitter().convert(chunk)) {
          if (line.startsWith('data:')) {
            var content = line.substring(5);
            if (content.startsWith(' ')) content = content.substring(1);
            if (content.startsWith('__STAGE__:')) {
              onStream?.call(content);
              continue;
            }
            if (content == '[DONE]') {
              break;
            }
            buffer.write(content);
            onStream?.call(content);
          }
        }
      }

      final finalText = buffer.toString();
      return ChatMessageModel.ai(chatId: userMessage.chatId, text: finalText);
    } on DioException catch (e) {
      if (CancelToken.isCancel(e)) rethrow;
      throw Exception('Error en la comunicación: ${e.message}');
    } catch (e) {
      throw Exception('Error inesperado: $e');
    }
  }

  Future<List<ClinicalCaseModel>> getClinicalCaseByUserId({
    required String userId,
  }) async {
    await Future.delayed(Duration(seconds: 1));

    List<ClinicalCaseModel> quizzes = [];
    return quizzes;
  }
}
