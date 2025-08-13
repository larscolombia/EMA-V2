import 'package:ema_educacion_medica_avanzada/app/actions/models/action_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/services/actions_service.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_message_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/data/api_clinical_case_data.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/interfaces/clinical_case_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_generate_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/data/local_questions_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';
import 'package:get/get.dart';

class ClinicalCasesServices {
  final _actionsService = Get.find<ActionsService>();
  final _apiClinicalCaseData = Get.find<ApiClinicalCaseData>();
  final _chatMessagesLocalData = Get.find<LocalChatMessageData>();
  final _localClinicalCaseData = Get.find<IClinicalCaseLocalData>();
  final _localQuestionsData = Get.find<LocalQuestionsData>();

  QuestionResponseModel? _initialQuestion;

  Future<ClinicalCaseModel?> getCaseById(String id) async {
    final where = 'uid = ?';
    final whereArgs = [id];

    final clinicalCase = await _localClinicalCaseData.getById(where, whereArgs);

    return clinicalCase;
  }

  Future<ClinicalCaseGenerateData> generateCase(
    ClinicalCaseModel temporalCase,
  ) async {
    final generated = await _apiClinicalCaseData.generateCase(temporalCase);

    await _localClinicalCaseData.insertOne(generated.clinicalCase);

    _initialQuestion = generated.question;

    return generated;
  }

  Future<List<QuestionResponseModel>> loadQuestionsByCaseId(
    String caseId,
  ) async {
    final where = 'quizId = ? AND parentType = ?';
    final whereArgs = [caseId, 'clinical_case'];

    final localQuestions = await _localQuestionsData.getItems(
      where: where,
      whereArgs: whereArgs,
    );

    if (localQuestions.isNotEmpty) {
      return localQuestions;
    }

    // Todo: implementar la funcionalidad remota, solicita el endpoint
    // final remoteQuestions = await _quizRemoteData.getQuestions(quiz);

    // await _localQuestionsData.insertMany(remoteQuestions);

    // return remoteQuestions;
    return [];
  }

  Future<List<ChatMessageModel>> loadMessageByCaseId(String caseId) async {
    final where = 'chatId = ?';
    final whereArgs = [caseId];

    final items = await _chatMessagesLocalData.getItems(
      where: where,
      whereArgs: whereArgs,
    );

    return items;
  }

  Future<QuestionResponseModel> _updateAnswer(
    QuestionResponseModel question,
  ) async {
    final where = 'id = ?';
    final whereArgs = [question.id];

    await _localQuestionsData.update(question, where, whereArgs);

    return question;
  }

  Future<void> insertQuestion(QuestionResponseModel question) async {
    // Se inserta la pregunta en la base de datos local
    await _localQuestionsData.insertOne(question);
  }

  Future<void> insertMessage(ChatMessageModel message) async {
    // Se inserta la pregunta en la base de datos local
    await _chatMessagesLocalData.insertOne(message);
  }

  Future<QuestionResponseModel> sendAnswer(
    QuestionResponseModel questionWithMessage,
  ) async {
    // Se envía la pregunta actual con la respuesta del usuario, para la respuesta del usuario
    // se utilizan las propiedades answer o answers (para futuras respuestas múltiples)
    // y adicionalmnente se envía la respuesta como mensaje usando el método resumeToInteractiveClinicalCase().
    // Se espera el feedback de la respuesta actual y una nueva pregunta.
    final feedBackAndNewQuestion = await _apiClinicalCaseData.sendAnswerMessage(
      questionWithMessage,
    );

    // Al actualizar, se refiere en actualizar la copia local de la pregunta, la respuesta y el feedback
    _updateAnswer(questionWithMessage);

    return feedBackAndNewQuestion;
  }

  Future<ChatMessageModel> sendMessage(ChatMessageModel userMessage) async {
    _chatMessagesLocalData.insertOne(userMessage);

    // usar la api
    final aiMessage = await _apiClinicalCaseData.sendMessage(userMessage);

    _chatMessagesLocalData.insertOne(aiMessage);

    return aiMessage;
  }

  Future<List<ChatMessageModel>> startAnalytical(
    ClinicalCaseModel clinicalCase,
  ) async {
    try {
      // Prompt mejorado que garantiza que la respuesta termine en pregunta y considere cierre con bibliografía
      const generatePrompt =
          '''Analiza este caso clínico y genera una respuesta que incluya:

1. Un análisis breve del caso
2. Los puntos clave a considerar
3. OBLIGATORIO: Termina con una pregunta específica y clara para guiar el análisis del estudiante

IMPORTANTE: 
- Tu respuesta DEBE terminar con una pregunta que invite al estudiante a continuar el análisis
- Ejemplos: "¿Cuál sería tu diagnóstico diferencial principal?" o "¿Qué exámenes complementarios solicitarías?"
- Al final del caso clínico (después de varios intercambios), incluye un punto de cierre con conclusiones y bibliografía relevante''';

      final userMessage = ChatMessageModel.user(
        chatId: clinicalCase.uid,
        text: generatePrompt,
      );

      // La información completa del caso ya se muestra en la vista mediante el
      // encabezado de anamnesis, por lo que no se envía como mensaje de chat.
      // final caseContentMessage = ChatMessageModel.ai(chatId: clinicalCase.uid, text: clinicalCase.textPlane);
      // final caseContentMessage = ChatMessageModel.ai(chatId: clinicalCase.uid, text: clinicalCase.textInMarkDown);

      final aiFirsQuestions = await _apiClinicalCaseData.sendMessage(
        userMessage,
      );

      // Solo se almacena la primera pregunta generada por la IA
      await _chatMessagesLocalData.insertOne(aiFirsQuestions);

      // Toma las primeras 10 palabras de la anamnesis para el título
      final shortTitle = clinicalCase.anamnesis.split(' ').take(10).join(' ');
      final title = clinicalCase.anamnesis;

      final action = ActionModel.clinicalCase(
        clinicalCase.userId, // Verificar
        clinicalCase.uid,
        shortTitle,
        title,
      );

      _actionsService.insertAction(action);

      // Se retorna únicamente la primera pregunta para iniciar la conversación
      return [aiFirsQuestions];
    } catch (e) {
      throw Exception('Error al generar el caso clínico');
    }
  }

  Future<QuestionResponseModel> startInteractive(
    ClinicalCaseModel clinicalCase,
  ) async {
    QuestionResponseModel? aiFirsQuestion = _initialQuestion;
    _initialQuestion = null;

    if (aiFirsQuestion == null) {
      final userFirstMessage = QuestionResponseModel.empty(
        quizId: clinicalCase.uid,
        parentType: 'clinical_case',
        message: 'Estoy listo para comenzar',
      );

      aiFirsQuestion = await _apiClinicalCaseData.sendAnswerMessage(
        userFirstMessage,
      );
    }

    // _localQuestionsData.insertOne(aiFirsQuestion);

    // Toma las primeras 10 palabras de la anamnesis para el título
    final shortTitle = clinicalCase.anamnesis.split(' ').take(10).join(' ');
    final title = clinicalCase.anamnesis;

    final action = ActionModel.clinicalCase(
      clinicalCase.userId, // Verificar
      clinicalCase.uid,
      shortTitle,
      title,
    );

    _actionsService.insertAction(action);

    return aiFirsQuestion;
  }
}
