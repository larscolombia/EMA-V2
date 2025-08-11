import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_generate_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';
import 'package:ema_educacion_medica_avanzada/core/api/api_service.dart';
import 'package:ema_educacion_medica_avanzada/core/logger/logger.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:get/get.dart';

class ApiClinicalCaseData {
  final Dio _dio = Get.find<ApiService>().dio;

  Future<ClinicalCaseGenerateData> generateCase(ClinicalCaseModel clinicalCase) async {
    try {
      final body = clinicalCase.toRequestBody();

      final url = clinicalCase.type == ClinicalCaseType.interactive ? '/casos-clinicos/interactivo' : '/caso-clinico';

      final response = await _dio.post(url, data: body);

      final generateClinicalCase = ClinicalCaseModel.fromApi(response.data);

      final completeClinicalCase = generateClinicalCase.copyWith(type: clinicalCase.type, uid: clinicalCase.uid, userId: clinicalCase.userId);

      QuestionResponseModel? firstQuestion;
      if (clinicalCase.type == ClinicalCaseType.interactive) {
        final data = response.data['data'] as Map<String, dynamic>;
        final questionsData = data['questions'] as Map<String, dynamic>?;
        
        if (questionsData != null) {
          final questionMap = {
            'id': 0, // No hay ID específico en la respuesta inicial
            'question': questionsData['texto'] ?? '',
            'answer': '', // No hay respuesta correcta en la pregunta inicial
            'type': questionsData['tipo'] ?? 'single_choice',
            'options': questionsData['opciones'] ?? [],
          };
          firstQuestion = QuestionResponseModel.fromClinicalCaseApi(quizId: clinicalCase.uid, feedback: '', questionMap: questionMap);
        }
      }

      try {
        final storage = const FlutterSecureStorage();
        await storage.write(key: 'interactive_case_thread_id', value: generateClinicalCase.threadId);
      } catch (_) {}

      return ClinicalCaseGenerateData(clinicalCase: completeClinicalCase, question: firstQuestion);
    } catch (e) {
      Logger.error(e.toString());
      throw Exception('No fue posible crear el caso clínico.');
    }
  }

  Future<QuestionResponseModel> sendAnswerMessage(QuestionResponseModel questionWithAnswer) async {
    final storage = const FlutterSecureStorage();
    final threadId = await storage.read(key: 'interactive_case_thread_id');

    final body = {'thread_id': threadId, 'mensaje': questionWithAnswer.message};

    Logger.objectValue('body_enviado_al_endpoint: /casos-clinicos/interactivo/conversar', body.toString());

    final response = await _dio.post('/casos-clinicos/interactivo/conversar', data: body);

    if (response.statusCode == 200) {
      final data = response.data['data'] ?? response.data;
      final feedback = data['feedback'] as String? ?? '';
      final questionMap = data['question'] as Map<String, dynamic>? ?? {};

      final newThreadId = data['thread_id'];
      if (newThreadId is String && newThreadId.isNotEmpty) {
        try {
          await storage.write(key: 'interactive_case_thread_id', value: newThreadId);
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

  Future<ChatMessageModel> sendMessage(ChatMessageModel userMessage) async {
    final storage = const FlutterSecureStorage();
    final threadId = await storage.read(key: 'interactive_case_thread_id');

    final body = {'thread_id': threadId, 'mensaje': userMessage.text};

    final response = await _dio.post('/casos-clinicos/conversar', data: body);

    if (response.statusCode == 200) {
      final chatId = userMessage.chatId;
      final text = response.data['respuesta']["text"] as String;
      return ChatMessageModel.ai(chatId: chatId, text: text);
    } else {
      throw Exception('Error al enviar mensaje');
    }
  }

  Future<List<ClinicalCaseModel>> getClinicalCaseByUserId({required String userId}) async {
    await Future.delayed(Duration(seconds: 1));

    List<ClinicalCaseModel> quizzes = [];
    return quizzes;
  }
}
