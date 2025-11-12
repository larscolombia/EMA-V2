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
    print('[LOAD_MSGS] 🔍 Cargando mensajes para caso: $caseId');

    final where = 'chatId = ?';
    final whereArgs = [caseId];

    final items = await _chatMessagesLocalData.getItems(
      where: where,
      whereArgs: whereArgs,
    );

    print('[LOAD_MSGS] 📦 Mensajes cargados: ${items.length}');
    print('[LOAD_MSGS] 📊 Tipo: ${items.runtimeType}');

    if (items.isNotEmpty) {
      print(
        '[LOAD_MSGS] 📝 Primer mensaje - AI: ${items.first.aiMessage}, Length: ${items.first.text.length}',
      );
      print(
        '[LOAD_MSGS] 📝 Último mensaje - AI: ${items.last.aiMessage}, Length: ${items.last.text.length}',
      );
    }

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

  Future<void> updateQuestionEvaluation(QuestionResponseModel question) async {
    // Actualiza la evaluación de una pregunta en la base de datos local
    final where = 'id = ?';
    final whereArgs = [question.id];
    await _localQuestionsData.update(question, where, whereArgs);
  }

  Future<void> insertMessage(ChatMessageModel message) async {
    print('[INSERT_MSG] 💾 Insertando mensaje en BD...');
    print('[INSERT_MSG] 📝 ID: ${message.uid}');
    print('[INSERT_MSG] 📋 ChatId: ${message.chatId}');
    print('[INSERT_MSG] 🤖 AI Message: ${message.aiMessage}');
    print('[INSERT_MSG] 📝 Longitud: ${message.text.length} chars');
    print('[INSERT_MSG] 🎨 Format: ${message.format}');

    // Se inserta la pregunta en la base de datos local
    await _chatMessagesLocalData.insertOne(message);
    print('[INSERT_MSG] ✅ Mensaje insertado exitosamente en BD');
  }

  Future<Map<String, dynamic>> sendAnswer(
    QuestionResponseModel questionWithMessage,
  ) async {
    // Se envía la pregunta actual con la respuesta del usuario, para la respuesta del usuario
    // se utilizan las propiedades answer o answers (para futuras respuestas múltiples)
    // y adicionalmnente se envía la respuesta como mensaje usando el método resumeToInteractiveClinicalCase().
    // Se espera el feedback de la respuesta actual y una nueva pregunta.
    final result = await _apiClinicalCaseData.sendAnswerMessage(
      questionWithMessage,
    );

    final feedBackAndNewQuestion =
        result['nextQuestion'] as QuestionResponseModel;
    final previousIsCorrect = result['previousIsCorrect'] as bool?;

    // Al actualizar, se refiere en actualizar la copia local de la pregunta, la respuesta y el feedback
    _updateAnswer(questionWithMessage);

    return {
      'nextQuestion': feedBackAndNewQuestion,
      'previousIsCorrect': previousIsCorrect,
    };
  }

  Future<ChatMessageModel> sendMessage(
    ChatMessageModel userMessage, {
    void Function(String token)? onStream,
  }) async {
    _chatMessagesLocalData.insertOne(userMessage);

    // usar la api con streaming de etapas opcional
    final aiMessage = await _apiClinicalCaseData.sendMessage(
      userMessage,
      onStream: onStream,
    );

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
    print('[SERVICE_EVAL] 🔄 Iniciando generateAnalyticalEvaluation...');
    print('[SERVICE_EVAL] 📋 Caso: ${clinicalCase.uid}');

    // Cargar todos los mensajes previos del caso
    print('[SERVICE_EVAL] 🔍 Cargando historial de mensajes...');
    final historyRaw = await loadMessageByCaseId(clinicalCase.uid);
    print('[SERVICE_EVAL] 📦 Mensajes raw: ${historyRaw.length}');

    // Casting explícito para asegurar tipo correcto
    final history = List<ChatMessageModel>.from(historyRaw);
    print('[SERVICE_EVAL] ✅ Casting exitoso: ${history.length} mensajes');

    // Filtrar solo intervenciones del usuario reales (evitar prompts internos)
    final userTurns = history.where((m) => !m.aiMessage).toList();
    print('[SERVICE_EVAL] 👤 Intervenciones de usuario: ${userTurns.length}');

    if (userTurns.isEmpty) {
      print('[SERVICE_EVAL] ⚠️ Sin intervenciones del usuario');
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

    print(
      '[SERVICE_EVAL] 📝 Usando últimas ${lastTurns.length} intervenciones',
    );

    final buffer = StringBuffer();
    for (int i = 0; i < lastTurns.length; i++) {
      final raw = lastTurns[i].text.replaceAll('\n', ' ');
      final truncated = raw.length > 280 ? raw.substring(0, 280) + '…' : raw;
      buffer.writeln('${i + 1}. $truncated');
    }

    print('[SERVICE_EVAL] 📄 Creando prompt de evaluación...');
    final evaluationPrompt = ChatMessageModel.user(
      chatId: clinicalCase.uid,
      // IMPORTANTE: NO guardar este mensaje en BD, solo usarlo para generar evaluación
      text:
          '[[HIDDEN_EVAL_PROMPT]] Genera EVALUACIÓN FINAL DETALLADA del desempeño sobre el caso clínico. '
          'Usa SOLO intervenciones listadas abajo.'
          '\n\nFORMATO MARKDOWN con estas secciones (línea en blanco ANTES y DESPUÉS de cada encabezado):'
          '\n\n# Resumen Clínico\n(2-4 frases: caso y cómo lo abordó el usuario)'
          '\n\n## Desempeño Global\n(2-3 frases: razonamiento, estructura, priorización)'
          '\n\n## Fortalezas\n- (bullets de fortalezas)'
          '\n\n## Áreas de Mejora\n- (bullets específicas)'
          '\n\n## Recomendaciones\n- (bullets concretas)'
          '\n\n## Errores Críticos\n- (detallar o "Ninguno identificado")'
          '\n\n## Puntuación\nPuntuación: NN/100 – justificación (1-2 líneas)'
          '\n\n## Referencias\n- (2-4 fuentes: Autor (año). Título/Guía.)'
          '\n\nSepara secciones con línea en blanco. NO preguntas al final.'
          '\n\nIntervenciones:'
          '\n${buffer.toString()}',
    );

    print('[SERVICE_EVAL] 📤 Enviando mensaje al backend...');
    print(
      '[SERVICE_EVAL] 📊 Longitud del prompt: ${evaluationPrompt.text.length} chars',
    );

    // NO guardar el prompt oculto en BD - solo enviar al backend y guardar la respuesta
    final aiEvaluation = await _apiClinicalCaseData.sendMessage(
      evaluationPrompt,
    );

    print('[SERVICE_EVAL] 📥 Respuesta recibida del backend');
    print('[SERVICE_EVAL] 📝 ID: ${aiEvaluation.uid}');
    print('[SERVICE_EVAL] 📝 Longitud: ${aiEvaluation.text.length} chars');
    print('[SERVICE_EVAL] 📝 Format: ${aiEvaluation.format}');
    print(
      '[SERVICE_EVAL] 📝 Preview (100 chars): ${aiEvaluation.text.substring(0, aiEvaluation.text.length > 100 ? 100 : aiEvaluation.text.length)}',
    );

    // Guardar solo la evaluación AI (sin el prompt oculto)
    print('[SERVICE_EVAL] 💾 Guardando evaluación en BD...');
    await insertMessage(aiEvaluation);
    print('[SERVICE_EVAL] ✅ Evaluación guardada exitosamente');

    return aiEvaluation;
  }

  /// DEPRECATED: La evaluación final es generada por el backend.
  /// Este método se mantiene solo por compatibilidad hacia atrás.
  /// No llamar desde nuevo código.
  @Deprecated('Use backend evaluation instead')
  Future<ChatMessageModel> generateInteractiveEvaluation(
    ClinicalCaseModel clinicalCase,
    List<QuestionResponseModel> questions,
  ) async {
    // Solo retorna un mensaje placeholder
    return ChatMessageModel.ai(
      chatId: clinicalCase.uid,
      text: 'Evaluación generada por el servidor.',
    );
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

      final result = await _apiClinicalCaseData.sendAnswerMessage(
        userFirstMessage,
      );
      aiFirsQuestion = result['nextQuestion'] as QuestionResponseModel;
      // No hay evaluación para el mensaje inicial, así que ignoramos previousIsCorrect
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
