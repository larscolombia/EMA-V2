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

      // ===== DEBUG: VER RESPUESTA COMPLETA DEL BACKEND (INICIO DE CASO) =====
      if (clinicalCase.type == ClinicalCaseType.interactive) {
        print(
          '╔═══════════════════════════════════════════════════════════════════',
        );
        print('║ RESPUESTA COMPLETA DEL BACKEND /casos-interactivos/iniciar');
        print(
          '╠═══════════════════════════════════════════════════════════════════',
        );
        print('║ Status Code: ${response.statusCode}');
        print(
          '╟───────────────────────────────────────────────────────────────────',
        );
        print('║ Response.data COMPLETO:');
        print('║ ${const JsonEncoder.withIndent('  ').convert(response.data)}');
        print(
          '╚═══════════════════════════════════════════════════════════════════',
        );
      }

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

  Future<Map<String, dynamic>> sendAnswerMessage(
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

    // ===== DEBUG: VER RESPUESTA COMPLETA DEL BACKEND =====
    print(
      '╔═══════════════════════════════════════════════════════════════════',
    );
    print('║ RESPUESTA COMPLETA DEL BACKEND /casos-interactivos/mensaje');
    print(
      '╠═══════════════════════════════════════════════════════════════════',
    );
    print('║ Status Code: ${response.statusCode}');
    print(
      '╟───────────────────────────────────────────────────────────────────',
    );
    print('║ Response.data COMPLETO:');
    print('║ ${const JsonEncoder.withIndent('  ').convert(response.data)}');
    print(
      '╚═══════════════════════════════════════════════════════════════════',
    );

    if (response.statusCode == 200) {
      final data = response.data['data'] ?? response.data;
      final feedback = data['feedback'] as String? ?? '';

      // ===== DEBUG: EXTRAER Y MOSTRAR FEEDBACK =====
      print(
        '╔═══════════════════════════════════════════════════════════════════',
      );
      print('║ FEEDBACK EXTRAÍDO DE LA RESPUESTA');
      print(
        '╠═══════════════════════════════════════════════════════════════════',
      );
      print('║ Longitud del feedback: ${feedback.length} caracteres');
      print(
        '╟───────────────────────────────────────────────────────────────────',
      );
      print('║ CONTENIDO DEL FEEDBACK:');
      print('║ $feedback');
      print(
        '╚═══════════════════════════════════════════════════════════════════',
      );

      // Extraer evaluación de la respuesta anterior (si existe)
      final evaluation = data['evaluation'] as Map<String, dynamic>?;
      final isCorrect =
          evaluation != null && evaluation['is_correct'] != null
              ? (evaluation['is_correct'] as bool)
              : null;

      Logger.objectValue(
        'EVALUATION_DEBUG',
        'evaluation object: $evaluation, is_correct: $isCorrect',
      );

      final next =
          (data['next'] ?? const <String, dynamic>{}) as Map<String, dynamic>;
      final pregunta =
          (next['pregunta'] ?? const <String, dynamic>{})
              as Map<String, dynamic>;

      // ===== DEBUG: ESTRUCTURA DE LA SIGUIENTE PREGUNTA =====
      print(
        '╔═══════════════════════════════════════════════════════════════════',
      );
      print('║ ESTRUCTURA DE LA SIGUIENTE PREGUNTA');
      print(
        '╠═══════════════════════════════════════════════════════════════════',
      );
      print(
        '║ next object: ${const JsonEncoder.withIndent('  ').convert(next)}',
      );
      print(
        '╟───────────────────────────────────────────────────────────────────',
      );
      print(
        '║ pregunta object: ${const JsonEncoder.withIndent('  ').convert(pregunta)}',
      );
      print(
        '╚═══════════════════════════════════════════════════════════════════',
      );

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

      final nextQuestion = QuestionResponseModel.fromClinicalCaseApi(
        quizId: questionWithAnswer.quizId,
        feedback: feedback,
        questionMap: questionMap,
      );

      Logger.objectValue(
        'EVALUATION_DEBUG_RESULT',
        'Returning - previousIsCorrect: $isCorrect, nextQuestion: ${nextQuestion.question}',
      );

      // Retornar un mapa que contenga tanto la siguiente pregunta como la evaluación
      return {
        'nextQuestion': nextQuestion,
        'previousIsCorrect': isCorrect, // La evaluación de la pregunta anterior
      };
    } else {
      throw Exception('Error al enviar mensaje');
    }
  }

  Future<ChatMessageModel> sendMessage(
    ChatMessageModel userMessage, {
    CancelToken? cancelToken,
    void Function(String token)? onStream,
  }) async {
    print('[API_SEND] 🚀 Iniciando envío de mensaje...');
    print('[API_SEND] 📋 ChatId: ${userMessage.chatId}');
    print('[API_SEND] 📝 Longitud mensaje: ${userMessage.text.length} chars');
    print(
      '[API_SEND] 📝 Preview (100 chars): ${userMessage.text.substring(0, userMessage.text.length > 100 ? 100 : userMessage.text.length)}',
    );

    final storage = const FlutterSecureStorage();
    final threadId = await storage.read(key: 'interactive_case_thread_id');
    print('[API_SEND] 🔑 Thread ID: $threadId');

    final body = {'thread_id': threadId, 'mensaje': userMessage.text};

    try {
      _cancelTokens[threadId ?? ''] ??= CancelToken();
      final token = cancelToken ?? _cancelTokens[threadId ?? '']!;

      print('[API_SEND] 📤 Enviando POST a /casos-clinicos/conversar...');
      final response = await _dio.post<dio.ResponseBody>(
        '/casos-clinicos/conversar',
        data: body,
        cancelToken: token,
        options: Options(
          responseType: ResponseType.stream,
          headers: {'Accept': 'text/event-stream'},
          // Timeout extendido para casos clínicos: RAG + streaming puede tardar
          receiveTimeout: const Duration(
            minutes: 5,
          ), // 5 min para RAG + generación
          sendTimeout: const Duration(minutes: 1),
        ),
      );

      print('[API_SEND] 📥 Respuesta recibida, iniciando streaming...');
      _cancelTokens.remove(threadId);

      final bodyStream = response.data;
      if (bodyStream == null) {
        print('[API_SEND] ❌ ERROR: Respuesta de streaming vacía');
        throw Exception('Respuesta de streaming vacía');
      }

      final stream = utf8.decoder.bind(bodyStream.stream);
      final buffer = StringBuffer();
      bool isDone = false;
      int chunkCount = 0;

      await for (final chunk in stream) {
        chunkCount++;
        for (final line in const LineSplitter().convert(chunk)) {
          if (line.startsWith('data:')) {
            var content = line.substring(5);
            if (content.startsWith(' ')) content = content.substring(1);
            if (content.startsWith('__STAGE__:')) {
              print('[API_SEND] 🏷️ Stage: $content');
              onStream?.call(content);
              continue;
            }
            if (content == '[DONE]') {
              print('[API_SEND] ✅ Marcador [DONE] recibido');
              isDone = true;
              break;
            }
            buffer.write(content);
            onStream?.call(content);
          }
        }
        if (isDone) break;
      }

      print('[API_SEND] 📊 Total chunks procesados: $chunkCount');
      final finalText = buffer.toString();
      print('[API_SEND] 📝 Texto final - Longitud: ${finalText.length} chars');
      print(
        '[API_SEND] 📝 Preview (200 chars): ${finalText.substring(0, finalText.length > 200 ? 200 : finalText.length)}',
      );

      // Detectar automáticamente si el texto es Markdown estructurado (evaluación)
      final isMarkdown = _detectMarkdownFormat(finalText);
      print(
        '[API_SEND] 🎨 Formato detectado: ${isMarkdown ? "MARKDOWN" : "PLAIN"}',
      );

      final aiMessage = ChatMessageModel.ai(
        chatId: userMessage.chatId,
        text: finalText,
        format: isMarkdown ? MessageFormat.markdown : MessageFormat.plain,
      );
      print('[API_SEND] ✅ Mensaje AI creado - ID: ${aiMessage.uid}');

      return aiMessage;
    } on DioException catch (e) {
      print('[API_SEND] ❌ DioException: ${e.type} - ${e.message}');
      if (CancelToken.isCancel(e)) rethrow;
      throw Exception('Error en la comunicación: ${e.message}');
    } catch (e, stackTrace) {
      print('[API_SEND] ❌ ERROR: $e');
      print('[API_SEND] 📚 StackTrace: $stackTrace');
      throw Exception('Error inesperado: $e');
    }
  }

  /// Detecta si el texto contiene Markdown estructurado (evaluación)
  /// Busca indicadores: headers (#), secciones de evaluación, listas, etc.
  bool _detectMarkdownFormat(String text) {
    final lower = text.toLowerCase();

    // CRITERIO PRINCIPAL: Debe contener indicadores ESPECÍFICOS de evaluación
    // No basta con tener headers Markdown (## Análisis) - eso es chat normal

    // Indicador 1: Prompt de evaluación oculto (más confiable)
    if (text.contains('[[HIDDEN_EVAL_PROMPT]]')) {
      return true;
    }

    // Indicador 2: Headers de evaluación ESPECÍFICOS (# Resumen Clínico con #, no ##)
    final hasEvaluationHeader =
        text.contains(
          RegExp(
            r'^#\s+Resumen Clínico',
            multiLine: true,
            caseSensitive: false,
          ),
        ) ||
        text.contains(
          RegExp(
            r'^#\s+Resumen Clinico',
            multiLine: true,
            caseSensitive: false,
          ),
        );

    // Indicador 3: Secciones MÚLTIPLES de evaluación (no solo una)
    final sectionCount =
        [
          lower.contains('desempeño') || lower.contains('desempeno'),
          lower.contains('fortalezas'),
          lower.contains('áreas de mejora') ||
              lower.contains('areas de mejora'),
          lower.contains('recomendaciones'),
          lower.contains('errores críticos') ||
              lower.contains('errores criticos'),
          lower.contains('puntuación') || lower.contains('puntuacion'),
        ].where((hasSection) => hasSection).length;

    // Indicador 4: Longitud (evaluaciones son largas >2000 chars)
    final isVeryLong = text.length > 2000;

    // REGLA: Es evaluación si:
    // - Tiene header "# Resumen Clínico" (nivel 1, no 2) Y
    // - Tiene al menos 3 secciones de evaluación Y
    // - Es muy largo (>2000 chars)
    return hasEvaluationHeader && sectionCount >= 3 && isVeryLong;
  }

  Future<List<ClinicalCaseModel>> getClinicalCaseByUserId({
    required String userId,
  }) async {
    await Future.delayed(Duration(seconds: 1));

    List<ClinicalCaseModel> quizzes = [];
    return quizzes;
  }
}
