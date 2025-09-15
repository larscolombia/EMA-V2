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

  /// Detecta si hay casos similares al prompt del usuario
  Future<List<ClinicalCaseModel>> detectSimilarCases(
    int userId,
    String userPrompt, {
    int maxResults = 5,
  }) async {
    // Buscar casos similares basados en palabras clave
    final similarCases = await _localClinicalCaseData.findSimilarCases(
      userId,
      userPrompt,
      limit: maxResults,
    );

    return similarCases;
  }

  /// Genera un prompt anti-repetición basado en casos similares encontrados
  String generateAntiRepetitionContext(
    List<ClinicalCaseModel> similarCases,
    String originalPrompt,
  ) {
    if (similarCases.isEmpty) return originalPrompt;

    final buffer = StringBuffer();
    buffer.writeln(
      'CASOS PREVIOS A EVITAR (genera algo completamente diferente):',
    );

    for (int i = 0; i < similarCases.length; i++) {
      final case_ = similarCases[i];
      buffer.writeln('${i + 1}. ${case_.summary ?? case_.title}');
    }

    buffer.writeln('\nPROMPT ORIGINAL: $originalPrompt');
    buffer.writeln(
      '\nIMPORTANTE: Genera un caso clínico que sea temáticamente DIFERENTE a los listados arriba. Cambia especialidad, grupo etario, fisiopatología o enfoque clínico.',
    );

    return buffer.toString();
  }

  /// Obtiene estadísticas de casos para control de crecimiento
  Future<Map<String, int>> getCaseStatistics(int userId) async {
    final recentCases = await _localClinicalCaseData
        .getRecentCasesForSimilarity(
          userId,
          limit: 100, // Últimos 100 casos para estadísticas
        );

    final now = DateTime.now();
    final lastWeek = now.subtract(Duration(days: 7));
    final lastMonth = now.subtract(Duration(days: 30));

    int weeklyCount = 0;
    int monthlyCount = 0;

    for (final case_ in recentCases) {
      if (case_.createdAt.isAfter(lastWeek)) {
        weeklyCount++;
      }
      if (case_.createdAt.isAfter(lastMonth)) {
        monthlyCount++;
      }
    }

    return {
      'total': recentCases.length,
      'weekly': weeklyCount,
      'monthly': monthlyCount,
    };
  }

  /// Limpia casos antiguos si hay demasiados (TTL básico)
  Future<void> cleanupOldCases(int userId, {int maxCases = 200}) async {
    final allCases = await _localClinicalCaseData.getRecentCasesForSimilarity(
      userId,
      limit: maxCases * 2,
    );

    if (allCases.length > maxCases) {
      final casesToDelete = allCases.skip(maxCases).toList();
      for (final case_ in casesToDelete) {
        final where = 'uid = ?';
        final whereArgs = [case_.uid];
        await _localClinicalCaseData.delete(where: where, whereArgs: whereArgs);
        await _localClinicalCaseData.delete(where: where, whereArgs: whereArgs);
      }
      print('🧹 Limpieza: eliminados ${casesToDelete.length} casos antiguos');
    }
  }

  Future<ClinicalCaseGenerateData> generateCase(
    ClinicalCaseModel temporalCase,
  ) async {
    final generated = await _apiClinicalCaseData.generateCase(temporalCase);

    // Generar resumen automáticamente antes de guardar
    final summary = generated.clinicalCase.generateSummary();
    final caseWithSummary = generated.clinicalCase.copyWith(summary: summary);

    await _localClinicalCaseData.insertOne(caseWithSummary);

    _initialQuestion = generated.question;

    return ClinicalCaseGenerateData(
      clinicalCase: caseWithSummary,
      question: generated.question,
    );
  }

  /// Genera un caso con detección de similitud previa
  Future<ClinicalCaseGenerateData> generateCaseWithSimilarityCheck(
    ClinicalCaseModel temporalCase,
    String userPrompt,
  ) async {
    // 1. Limpieza automática de casos antiguos (cada 10 generaciones aprox.)
    final random = DateTime.now().millisecond % 10;
    if (random == 0) {
      await cleanupOldCases(temporalCase.userId);
    }

    // 2. Detectar casos similares
    final similarCases = await detectSimilarCases(
      temporalCase.userId,
      userPrompt,
      maxResults: 5,
    );

    // 3. Si hay casos similares, crear contexto anti-repetición
    ClinicalCaseModel caseToGenerate = temporalCase;
    if (similarCases.isNotEmpty) {
      // Generar prompt anti-repetición (por ahora solo para logging y futura extensión)
      final promptForDebug = generateAntiRepetitionContext(
        similarCases,
        userPrompt,
      );

      // Modificar el caso temporal con contexto mejorado
      // Nota: Esto requeriría modificar la API para aceptar contexto adicional
      // Por ahora, registramos la similitud en logs
      print('🔍 Casos similares detectados: ${similarCases.length}');
      for (final similar in similarCases) {
        print('  - ${similar.summary ?? similar.title}');
      }
      // También registramos el prompt anti-repetición generado para diagnóstico
      print('🛡️ Prompt anti-repetición generado (debug):');
      print(promptForDebug);
    }

    // 4. Generar el caso normalmente
    return await generateCase(caseToGenerate);
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

  /// Genera una evaluación final del desempeño del usuario en un caso analítico.
  /// Recolecta los mensajes (solo del usuario) guardados localmente para construir
  /// un prompt de evaluación. Retorna el mensaje AI con la evaluación.
  Future<ChatMessageModel> generateAnalyticalEvaluation(
    ClinicalCaseModel clinicalCase,
  ) async {
    // Cargar todos los mensajes previos del caso
    final history = await loadMessageByCaseId(clinicalCase.uid);
    // Filtrar solo intervenciones del usuario reales (evitar prompts internos)
    final userTurns = history.where((m) => !m.aiMessage).toList();
    if (userTurns.isEmpty) {
      // Mensaje AI directo indicando que no hay suficientes datos
      final aiEmpty = ChatMessageModel.ai(
        chatId: clinicalCase.uid,
        text:
            'No se pudo generar una evaluación detallada porque no se registraron intervenciones del usuario en este caso.',
      );
      await insertMessage(aiEmpty);
      return aiEmpty;
    }

    // Limitar a las últimas 15 intervenciones para controlar longitud
    final lastTurns =
        userTurns.length > 15
            ? userTurns.sublist(userTurns.length - 15)
            : userTurns;

    final buffer = StringBuffer();
    for (int i = 0; i < lastTurns.length; i++) {
      final raw = lastTurns[i].text.replaceAll('\n', ' ');
      final truncated = raw.length > 280 ? raw.substring(0, 280) + '…' : raw;
      buffer.writeln('${i + 1}. $truncated');
    }

    final evaluationPrompt = ChatMessageModel.user(
      chatId: clinicalCase.uid,
      // Prefijo especial para poder filtrar fácilmente en UI sin cambiar schema
      text: '[[HIDDEN_EVAL_PROMPT]] Genera una EVALUACIÓN FINAL DETALLADA del desempeño del usuario sobre el caso clínico. '
      text:
          'Genera una EVALUACIÓN FINAL DETALLADA del desempeño del usuario sobre el caso clínico. '
          'Usa SOLO las intervenciones listadas (no inventes nuevas). Devuelve en Markdown con EXACTAS estas secciones y encabezados:'
          '\n# Resumen Clínico (2-4 frases concisas)'
          '\n## Desempeño global (2-3 frases evaluando razonamiento clínico, estructura y priorización)'
          '\n## Fortalezas\n- (bullets)'
          '\n## Áreas de mejora\n- (bullets específicas, accionables)'
          '\n## Recomendaciones accionables\n- (bullets concretas de estudio / práctica)'
          '\n## Errores críticos\n- (o "Ninguno identificado")'
          '\n## Puntuación\nFormato: "Puntuación: NN/100 – breve justificación"'
          '\n## Referencias\n- 2-4 fuentes abreviadas (año, autor o institución, guía)'
          '\nAL FINAL: No formules preguntas, no abras nuevas líneas de conversación.'
          '\nIntervenciones del usuario para evaluar:'
          '\n${buffer.toString()}',
    );
    // Guardar el prompt de evaluación como intervención de usuario
    await insertMessage(evaluationPrompt);
    // Obtener respuesta AI usando el mismo hilo
    final aiEvaluation = await sendMessage(evaluationPrompt);
    return aiEvaluation;
  }

  /// Genera un resumen/evaluación final local para casos interactivos (tipo cuestionario secuencial)
  /// sin necesidad de llamar a la IA. Usa las preguntas almacenadas y sus respuestas.
  Future<ChatMessageModel> generateInteractiveEvaluation(
    ClinicalCaseModel clinicalCase,
    List<QuestionResponseModel> questions,
  ) async {
    // Filtrar preguntas respondidas válidas
    final answered =
        questions
            .where((q) => (q.userAnswer != null && q.userAnswer!.isNotEmpty))
            .toList();
    final total = answered.length;
    int correct = 0;
    for (final q in answered) {
      final ans = (q.answer ?? '').trim().toLowerCase();
      final user = (q.userAnswer ?? '').trim().toLowerCase();
      if (ans.isNotEmpty && ans == user) correct++;
    }
    final scorePct = total == 0 ? 0 : ((correct * 100) / total).round();

    String bulletLines = '';
    int idx = 1;
    for (final q in answered) {
      final ans = (q.answer ?? '').trim();
      final user = (q.userAnswer ?? '').trim();
      final isOk = ans.isNotEmpty && ans.toLowerCase() == user.toLowerCase();
      final feedback = (q.fit ?? '').trim();
      final truncatedFb =
          feedback.length > 220 ? feedback.substring(0, 220) + '…' : feedback;
      bulletLines +=
          '\n${idx}. ${q.question}\n  Tu respuesta: ${user.isEmpty ? '(vacía)' : user}'
          '  |  Correcta: ${ans.isEmpty ? 'N/D' : ans}'
          '  => ${isOk ? '✅' : '❌'}';
      if (truncatedFb.isNotEmpty) {
        bulletLines += '\n  Feedback: $truncatedFb';
      }
      idx++;
    }

    // Heurística simple de fortalezas / mejoras
    final fortalezas = <String>[];
    final mejoras = <String>[];
    if (scorePct >= 80) {
      fortalezas.add('Buen dominio general de los conceptos evaluados');
    } else if (scorePct >= 50) {
      fortalezas.add('Identificas parcialmente los conceptos clave');
    } else {
      mejoras.add('Repasar fundamentos teóricos antes de un nuevo intento');
    }
    if (correct < total) {
      mejoras.add('Revisar las preguntas marcadas con ❌');
    }
    if (fortalezas.isEmpty) fortalezas.add('Participación activa en el caso');
    if (mejoras.isEmpty)
      mejoras.add(
        'Profundizar en bibliografía recomendada para consolidar conocimiento',
      );

    final resumen =
        StringBuffer()
          ..writeln('Evaluación final interactiva')
          ..writeln('Puntuación: $correct / $total ($scorePct%)')
          ..writeln('\nFortalezas:')
          ..writeln(fortalezas.map((e) => '- $e').join('\n'))
          ..writeln('\nÁreas de mejora:')
          ..writeln(mejoras.map((e) => '- $e').join('\n'))
          ..writeln('\nDetalle de preguntas:')
          ..writeln(bulletLines)
          ..writeln('\nReferencias sugeridas (genéricas):')
          ..writeln('- Guías clínicas actualizadas de la especialidad')
          ..writeln(
            '- Revisión sistemática reciente en base de datos reconocida',
          );

    final evaluation = ChatMessageModel.ai(
      chatId: clinicalCase.uid,
      text: resumen.toString(),
    );
    await insertMessage(evaluation);
    return evaluation;
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
