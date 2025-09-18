import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/life_stage.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/sex_and_status.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/services/clinical_cases_services.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_generate_data.dart';
import 'package:ema_educacion_medica_avanzada/config/routes/app_pages.dart';
import 'package:ema_educacion_medica_avanzada/core/logger/logger.dart';
import 'package:ema_educacion_medica_avanzada/core/notify/notify.dart';
import 'package:ema_educacion_medica_avanzada/core/ui/ui_observer_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../profiles/profiles.dart';

class ClinicalCaseController extends GetxController
    with StateMixin<ClinicalCaseModel> {
  ScrollController? scrollController;
  // Dependencies (allow injection for tests). Use dynamic to accept lightweight fakes in tests.
  final dynamic clinicalCaseServive;
  final dynamic uiObserverService;
  final dynamic userService;
  final dynamic profileController;

  ClinicalCaseController({
    dynamic clinicalCaseServive,
    dynamic uiObserverService,
    dynamic userService,
    dynamic profileController,
  }) : clinicalCaseServive =
           clinicalCaseServive ?? Get.find<ClinicalCasesServices>(),
       uiObserverService = uiObserverService ?? Get.find<UiObserverService>(),
       userService = userService ?? Get.find<UserService>(),
       profileController = profileController ?? Get.find<ProfileController>();

  final Rx<ClinicalCaseModel?> currentCase = Rx(null);
  final messages = <ChatMessageModel>[].obs;
  final questions = <QuestionResponseModel>[].obs;

  final Rx<QuestionResponseModel?> currentQuestion = Rx(null);

  final isComplete = false.obs;
  final isTyping = false.obs; // New observable
  // Backend processing stage (for unified UI cues)
  final currentStage = ''.obs;
  final isFinalizingCase = false.obs; // Loading específico para finalización
  final analyticalAiTurns = 0.obs; // Cuenta respuestas IA en modo analítico
  static const int maxAnalyticalAiTurns = 15;
  final evaluationGenerated = false.obs; // Evita duplicar evaluación final
  final interactiveEvaluationGenerated = false.obs; // Para casos interactivos
  final evaluationInProgress = false.obs; // Evita dobles llamadas
  static const int maxInteractiveQuestions =
      15; // Límite de preguntas en modo interactivo
  final Rx<ChatMessageModel?> _pendingInteractiveSummary = Rx(
    null,
  ); // nuevo: guarda 'Resumen Final:' oculto

  bool get hasHiddenInteractiveSummary =>
      _pendingInteractiveSummary.value != null;

  ChatMessageModel? takeInteractiveSummary() {
    final m = _pendingInteractiveSummary.value;
    _pendingInteractiveSummary.value = null;
    if (m != null) {
      messages.add(m); // ahora sí lo mostramos
    }
    return m;
  }

  @override
  void onInit() {
    super.onInit();
    uiObserverService.isKeyboardVisible.listen((value) {
      if (value) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          _scrollToBottom();
        });
      }
    });
  }

  void generateCase({
    required ClinicalCaseType type,
    required LifeStage lifeStage,
    required SexAndStatus sexAndStatus,
    String userPrompt = '', // Nuevo parámetro para detección de similitud
  }) async {
    // Prevent multiple simultaneous calls
    if (isTyping.value) {
      print(
        '⚠️ [ClinicalCaseController] generateCase already in progress, ignoring duplicate call',
      );
      return;
    }

    try {
      // Reset state flags to avoid leftover completion blocking input
      isComplete.value = false;
      evaluationGenerated.value = false;
      interactiveEvaluationGenerated.value = false;
      analyticalAiTurns.value = 0;
      if (!profileController.canCreateMoreClinicalCases()) {
        Get.snackbar(
          'Límite alcanzado',
          'Has alcanzado el límite de casos clínicos en tu plan actual. Actualiza tu plan para crear más casos clínicos.',
          snackPosition: SnackPosition.TOP,
          backgroundColor: Colors.orange,
          colorText: Colors.white,
          duration: const Duration(seconds: 5),
          mainButton: TextButton(
            onPressed: () => Get.toNamed(Routes.subscriptions.name),
            child: const Text(
              'Actualizar Plan',
              style: TextStyle(color: Colors.white),
            ),
          ),
        );
        Get.toNamed(Routes.home.name, preventDuplicates: true);
        return;
      }

      isTyping.value = true; // Lock navigation while generating the case
      print(
        '🎯 [ClinicalCaseController] Starting case generation: ${type.name}',
      );

      final userId = userService.currentUser.value.id;

      final temporalCase = ClinicalCaseModel.generate(
        userId: userId,
        type: type,
        lifeStage: lifeStage,
        sexAndStatus: sexAndStatus,
      );

      messages.clear();
      questions.clear();
      currentQuestion.value = null;

      change(temporalCase, status: RxStatus.loading());

      final route =
          type == ClinicalCaseType.analytical
              ? Routes.clinicalCaseAnalytical.path(temporalCase.uid)
              : Routes.clinicalCaseInteractive.path(temporalCase.uid);

      Get.offAndToNamed(route);

      _startClinicalCase(temporalCase, userPrompt);

      // Quota consumption now centralized in backend (analytical_generate / interactive_generate flows)
    } catch (e) {
      print('❌ [ClinicalCaseController] Error in generateCase: $e');
      change(
        null,
        status: RxStatus.error('No se pudo generar el caso clínico'),
      );
      isTyping.value = false; // Unlock navigation on error
      Logger.error(e.toString());
    }
  }

  Future<void> _startClinicalCase(
    ClinicalCaseModel temporalCase, [
    String userPrompt = '',
  ]) async {
    try {
      // Usar detección de similitud si se proporciona un prompt del usuario
      ClinicalCaseGenerateData generated;
      if (userPrompt.isNotEmpty) {
        generated = await clinicalCaseServive.generateCaseWithSimilarityCheck(
          temporalCase,
          userPrompt,
        );
      } else {
        generated = await clinicalCaseServive.generateCase(temporalCase);
      }

      final clinicalCase = generated.clinicalCase;

      Logger.mini('_startClinicalCase');
      Logger.objectValue('clinicalCase.textPlane', clinicalCase.textPlane);

      if (clinicalCase.type == ClinicalCaseType.analytical) {
        final newMessages = await clinicalCaseServive.startAnalytical(
          clinicalCase,
        );
        _insertMessages(newMessages);
        analyticalAiTurns.value = newMessages.length; // usualmente 1
      } else {
        final newQuestion = await clinicalCaseServive.startInteractive(
          clinicalCase,
        );

        Logger.objectValue('newQuestion', newQuestion.toString());

        final aiQuestionMessage = ChatMessageModel.ai(
          chatId: newQuestion.quizId,
          text: newQuestion.toAiQuestionMessage(),
        );

        Logger.objectValue('aiQuestionMessage', aiQuestionMessage.toString());

        _insertMessages([aiQuestionMessage]);
        clinicalCaseServive.insertMessage(aiQuestionMessage);

        // generar una nueva pregunta a partir de la respuesta de la ia
        // debe tener el mismo id del mensaje con la pregunta
        final questionFromMessage = newQuestion.copyWith(
          id: aiQuestionMessage.uid,
          parentType: 'clinical_case',
        );

        Logger.objectValue(
          'questionFromMessage',
          questionFromMessage.toString(),
        );

        // actualizar los rx de pregunta actual y pregunta pendiente
        currentQuestion.value = questionFromMessage;

        // guardar los mensajes en la base de datos local
        _insertQuestions([questionFromMessage]);
        clinicalCaseServive.insertQuestion(questionFromMessage);
      }

      currentCase.value = clinicalCase;

      change(clinicalCase, status: RxStatus.success());
      isTyping.value = false; // Unlock navigation when generation ends
    } catch (e) {
      currentCase.value = null;
      change(
        null,
        status: RxStatus.error('No se pudo generar el caso clínico'),
      );
      isTyping.value = false; // Ensure unlock on error
      Logger.error(e.toString());
    }
  }

  void _insertQuestions(List<QuestionResponseModel> questionsToInsert) {
    questions.addAll(questionsToInsert);
    _scrollToBottom();
  }

  void _insertMessages(List<ChatMessageModel> messagesToInsert) {
    messages.addAll(messagesToInsert);
    _scrollToBottom();
  }

  void _loadCaseAndMessages(String caseId) async {
    // Clear previous data before loading the selected case
    messages.clear();
    questions.clear();
    currentQuestion.value = null;

    final clinicalCase = await clinicalCaseServive.getCaseById(caseId);

    if (clinicalCase == null) {
      currentCase.value = null;
      change(null, status: RxStatus.empty());
    } else {
      final loadedMessages = await clinicalCaseServive.loadMessageByCaseId(
        caseId,
      );
      final loadedQuestions = await clinicalCaseServive.loadQuestionsByCaseId(
        caseId,
      );

      currentCase.value = clinicalCase;
      change(clinicalCase, status: RxStatus.success());

      WidgetsBinding.instance.addPostFrameCallback((_) {
        _insertMessages(loadedMessages);
        _insertQuestions(loadedQuestions);
        currentQuestion.value =
            loadedQuestions.isNotEmpty ? loadedQuestions.first : null;
      });
    }
  }

  Future<void> sendAnswer({
    required QuestionResponseModel question,
    required AnswerModel userAnswer,
  }) async {
    try {
      isTyping.value = true; // Set typing to true at the beginning

      final questionWithAnswer = question.copyWith(
        parentType: 'clinical_case',
        userAnswer: userAnswer.answer,
        userAnswers: userAnswer.answers,
      );

      Logger.mini('sendAnswer');
      Logger.objectValue('questionWithAnswer', questionWithAnswer.toString());

      // Se hace en este orden porque el método utiliza los valores
      // de la pregunta y la respuesta del usuario para generar el mensaje
      final questionWithMessage = questionWithAnswer.copyWith(
        message: questionWithAnswer.toUserAnswerMessage(),
      );

      Logger.objectValue('questionWithMessage', questionWithMessage.toString());

      // Update the question in the questions list
      final questionIndex = questions.indexWhere((q) => q.id == question.id);
      if (questionIndex >= 0) {
        questions[questionIndex] = questionWithMessage;
      }

      // Keep currentQuestion reference (don't set to null), just update it
      // This keeps the UI showing the question while waiting
      currentQuestion.value = questionWithMessage;

      // Generar un mensaje a partir de la respuesta del usuario
      final userMessage = ChatMessageModel.user(
        chatId: questionWithAnswer.quizId,
        text: questionWithMessage.toUserAnswerMessage(),
      );

      Logger.objectValue('userMessage', userMessage.toString());

      _insertMessages([userMessage]);
      clinicalCaseServive.insertMessage(userMessage);

      // envía la respuesta al servidor y actualiza la respuesta en local
      final feedBackAndNewQuestion = await clinicalCaseServive.sendAnswer(
        questionWithMessage,
      );

      Logger.objectValue(
        'feedBackAndNewQuestion',
        feedBackAndNewQuestion.toString(),
      );

      // Generar un mensaje de ia a partir de la respuesta de la ia
      final aiFeedBackMessage = ChatMessageModel.ai(
        chatId: questionWithAnswer.quizId,
        text: feedBackAndNewQuestion.fit!,
      );

      Logger.objectValue('aiFeedBackMessage', aiFeedBackMessage.toString());

      _insertMessages([aiFeedBackMessage]);
      clinicalCaseServive.insertMessage(aiFeedBackMessage);

      // en ese caso se trata del último comentario de la IA
      // para cerrar el caso clínico.
      if (feedBackAndNewQuestion.question == '') {
        isComplete.value = true;
        // Capturar mensaje de resumen final si llegó (prefijo 'Resumen Final:') y guardarlo oculto
        final summaryText = feedBackAndNewQuestion.fit ?? '';
        if (summaryText.startsWith('Resumen Final:')) {
          _pendingInteractiveSummary.value = ChatMessageModel.ai(
            chatId: questionWithAnswer.quizId,
            text: summaryText,
          );
        }
        interactiveEvaluationGenerated.value = false; // aún no mostrado
        isTyping.value = false; // Make sure to set typing to false
        return;
      } else {
        // Chequear límite antes de agregar una nueva pregunta
        final totalQuestionsBefore =
            questions.length; // ya incluye la actual respondida
        if (totalQuestionsBefore >= maxInteractiveQuestions) {
          // Marcar fin forzado: no se agrega nueva pregunta
          isComplete.value = true;
          final summaryText = feedBackAndNewQuestion.fit ?? '';
          if (summaryText.startsWith('Resumen Final:')) {
            _pendingInteractiveSummary.value = ChatMessageModel.ai(
              chatId: questionWithAnswer.quizId,
              text: summaryText,
            );
          }
          interactiveEvaluationGenerated.value = false;
          isTyping.value = false;
          _scrollToBottom();
          return;
        }
        final aiQuestionMessage = ChatMessageModel.ai(
          chatId: questionWithAnswer.quizId,
          text: feedBackAndNewQuestion.toAiQuestionMessage(),
        );

        Logger.objectValue('aiQuestionMessage', aiQuestionMessage.toString());

        _insertMessages([aiQuestionMessage]);
        clinicalCaseServive.insertMessage(aiQuestionMessage);

        // generar una nueva pregunta a partir de la respuesta de la ia
        // debe tener el mismo id del mensaje con la pregunta
        final questionForMessage = feedBackAndNewQuestion.copyWith(
          id: aiQuestionMessage.uid,
          parentType: 'clinical_case',
        );

        Logger.objectValue('questionForMessage', questionForMessage.toString());

        // Set the new question as current question
        currentQuestion.value = questionForMessage;

        // Add to questions list if not already there
        if (!questions.any((q) => q.id == questionForMessage.id)) {
          questions.add(questionForMessage);
        }

        // guardar los mensajes en la base de datos local
        clinicalCaseServive.insertQuestion(questionForMessage);
      }

      isTyping.value = false; // Set typing to false when done

      _scrollToBottom();
    } catch (e) {
      Notify.snackbar(
        'Casos clínicos',
        'Falló al intentar enviar la respuesta al servidor.',
        NotifyType.error,
      );
      isTyping.value = false; // Make sure typing is false on error
    }
  }

  Future<void> sendMessage(String userText) async {
    final String cleanUserText = userText.trim();
    ClinicalCaseModel? clinicalCase = value;

    try {
      if (cleanUserText.isEmpty) {
        return;
      }
      // Bloquear más mensajes si ya se completó el caso analítico
      if (isComplete.value &&
          clinicalCase?.type == ClinicalCaseType.analytical) {
        return;
      }
      if (clinicalCase == null) {
        throw Exception('Se perdió la conexción del caso clínico');
      }

      final userMessage = ChatMessageModel.user(
        chatId: clinicalCase.uid,
        text: cleanUserText,
      );

      messages.add(userMessage);
      _scrollToBottom();

      isTyping.value = true; // Show typing indicator

      // Pasar callback onStream para recibir marcadores de etapa SSE
      final aiMessage = await clinicalCaseServive.sendMessage(
        userMessage,
        onStream: (token) {
          try {
            if (token.startsWith('__STAGE__:')) {
              final stage = token.split(':')[1];
              currentStage.value = stage;
            }
          } catch (_) {}
        },
      ); // Await AI response
      messages.add(aiMessage);
      // Limpiar estado de etapa al terminar
      currentStage.value = '';
      if (clinicalCase.type == ClinicalCaseType.analytical) {
        analyticalAiTurns.value += 1;
        // Si alcanzó el máximo, generar cierre automático
        if (analyticalAiTurns.value >= maxAnalyticalAiTurns &&
            !isComplete.value) {
          await _finalizeAnalyticalCase(clinicalCase);
        }
        // Heurística: si el modelo ya entregó retroalimentación completa (bibliografía / resumen / diagnóstico) antes del máximo
        else if (!isComplete.value &&
            analyticalAiTurns.value > 1 &&
            _looksLikeAnalyticalClosure(aiMessage.text)) {
          _completeAnalyticalEarly(clinicalCase, aiMessage.text);
        }
      }

      _scrollToBottom();

      isTyping.value = false; // Hide typing indicator
    } catch (e) {
      Notify.snackbar('Casos Clínicos', e.toString(), NotifyType.error);
      Logger.error(
        e.toString(),
        className: 'ChatController',
        methodName: 'sendMessage',
        meta: 'userText: $userText',
      );
      isTyping.value = false; // Ensure indicator is hidden on error
    }
  }

  /// Genera un mensaje de evaluación final explícita en modo analítico
  /// incluso después de haber marcado el caso como completo.
  Future<void> generateFinalEvaluation() async {
    final clinicalCase = currentCase.value;
    if (clinicalCase == null ||
        clinicalCase.type != ClinicalCaseType.analytical) {
      return;
    }
    if (evaluationInProgress.value || evaluationGenerated.value) {
      return;
    }
    try {
      evaluationInProgress.value = true;
      isTyping.value = true;
      // Navegar primero para mostrar loader blanco
      Get.offAndToNamed(Routes.clinicalCaseEvaluation.path(clinicalCase.uid));
      // Generar evaluación (puede tardar, la vista muestra loader)
      final evaluationMessage = await clinicalCaseServive
          .generateAnalyticalEvaluation(clinicalCase);
      messages.add(evaluationMessage);
      _scrollToBottom();
      evaluationGenerated.value = true;
    } catch (e) {
      Logger.error('Error al generar evaluación analítica: $e');
      Notify.snackbar(
        'Casos clínicos',
        'No se pudo generar la evaluación final.',
        NotifyType.error,
      );
    } finally {
      evaluationInProgress.value = false;
      isTyping.value = false;
    }
  }

  bool _looksLikeAnalyticalClosure(String text) {
    final lower = text.toLowerCase();
    final hasBibliography = lower.contains('bibliograf');
    final hasSummary =
        lower.contains('resumen') ||
        lower.contains('conclusión') ||
        lower.contains('conclusion');
    final hasDiagnosis =
        lower.contains('diagnóstico') || lower.contains('diagnostico');
    final mentionsPlan = lower.contains('plan') || lower.contains('manejo');
    final looksFinalPhrase =
        lower.contains('fin del caso') || lower.contains('caso finalizado');
    // No parece estar solicitando más (ausencia de signo de interrogación múltiple y palabras de pregunta al final)
    final questionMarks = RegExp(r'[?¿]').allMatches(lower).length;
    final endsWithQuestion =
        lower.trim().endsWith('?') || lower.trim().endsWith('¿');
    final hasPromptForMore =
        lower.contains('otra pregunta') ||
        lower.contains('algo más') ||
        lower.contains('algo mas');
    final closureSignals =
        (hasBibliography && hasDiagnosis) ||
        looksFinalPhrase ||
        (hasSummary && hasDiagnosis && mentionsPlan);
    return closureSignals &&
        questionMarks < 2 &&
        !endsWithQuestion &&
        !hasPromptForMore;
  }

  /// Indica si debemos ofrecer al usuario un botón para finalizar el caso analítico.
  bool get shouldOfferAnalyticalFinalize {
    final caseModel = currentCase.value;
    if (caseModel == null || caseModel.type != ClinicalCaseType.analytical)
      return false;
    if (isComplete.value ||
        evaluationInProgress.value ||
        evaluationGenerated.value)
      return false;
    // Si ya hubo suficientes turnos de la IA
    if (analyticalAiTurns.value >= 4) return true; // umbral configurable
    // O si el último mensaje AI parece de cierre
    for (final m in messages.reversed) {
      if (m.aiMessage) {
        return _looksLikeAnalyticalClosure(m.text);
      }
    }
    return false;
  }

  /// Finaliza caso analítico desde la UI (botón del usuario)
  Future<void> finalizeAnalyticalFromUser() async {
    final caseModel = currentCase.value;
    if (caseModel == null || caseModel.type != ClinicalCaseType.analytical)
      return;
    if (isComplete.value ||
        evaluationInProgress.value ||
        isFinalizingCase.value)
      return;

    isFinalizingCase.value = true;
    try {
      await _finalizeAnalyticalCase(caseModel);
    } finally {
      isFinalizingCase.value = false;
    }
  }

  void _completeAnalyticalEarly(ClinicalCaseModel clinicalCase, String aiText) {
    isComplete.value = true;
    // Añadir mensaje explícito de cierre si el texto del modelo no contiene una marca clara
    final lower = aiText.toLowerCase();
    if (!lower.contains('fin del caso')) {
      messages.add(
        ChatMessageModel.ai(
          chatId: clinicalCase.uid,
          text:
              'Fin del caso clínico. Si deseas, puedes iniciar un nuevo caso para continuar practicando.',
        ),
      );
    }
    generateFinalEvaluation();
  }

  Future<void> _finalizeAnalyticalCase(ClinicalCaseModel clinicalCase) async {
    try {
      isTyping.value = true;
      // Prompt para cierre: resume hallazgos, diagnóstico probable, diferenciales, plan inicial y bibliografía.
      const closingPrompt =
          'Genera un resumen final estructurado del caso: '
          '1) Resumen clínico conciso 2) Diagnóstico más probable y diferenciales clave 3) Justificación clínica '
          '4) Plan diagnóstico y terapéutico inicial 5) Errores comunes a evitar 6) Bibliografía (formato breve). '
          'No hagas más preguntas. Marca el final claramente.';

      // Crear mensaje de usuario interno sin agregarlo a la UI visible
      final userClosingMessage = ChatMessageModel.user(
        chatId: clinicalCase.uid,
        text: closingPrompt,
      );

      // Guardar en base de datos pero NO añadir a messages visibles
      clinicalCaseServive.insertMessage(userClosingMessage);

      final aiClosing = await clinicalCaseServive.sendMessage(
        userClosingMessage,
      );
      messages.add(aiClosing);
      isComplete.value = true;
      await generateFinalEvaluation();
    } catch (e) {
      // Si falla el cierre, no bloquear al usuario (podría intentar nuevamente manualmente)
      Logger.error('Error al generar cierre analítico: $e');
    } finally {
      isTyping.value = false;
      _scrollToBottom();
    }
  }

  void _scrollToBottom() {
    if (scrollController == null || scrollController!.hasClients == false) {
      return;
    }

    WidgetsBinding.instance.addPostFrameCallback((_) {
      try {
        for (final position in scrollController!.positions) {
          position.animateTo(
            position.maxScrollExtent,
            duration: const Duration(milliseconds: 300),
            curve: Curves.easeOut,
          );
        }
      } catch (e) {
        // Ignore scroll errors
      }
    });
  }

  void setScrollController(ScrollController? controller) {
    scrollController = controller;
  }

  void showChat(String caseId) async {
    try {
      final clinicalCase = await clinicalCaseServive.getCaseById(caseId);

      if (clinicalCase != null &&
          clinicalCase.type == ClinicalCaseType.analytical) {
        Get.offAndToNamed(Routes.clinicalCaseAnalytical.path(caseId));
      } else {
        Get.offAndToNamed(Routes.clinicalCaseInteractive.path(caseId));
      }

      WidgetsBinding.instance.addPostFrameCallback((_) {
        _loadCaseAndMessages(caseId);
      });
    } catch (e) {
      currentCase.value = null;
      change(
        null,
        status: RxStatus.error('Ocurrió un error al cargar el caso clínico'),
      );
      Logger.error(e.toString());
    }
  }

  /// Obtiene estadísticas de casos del usuario actual
  Future<Map<String, int>> getUserCaseStatistics() async {
    final userId = userService.currentUser.value.id;
    return await clinicalCaseServive.getCaseStatistics(userId);
  }

  /// Detecta casos similares para mostrar al usuario antes de generar
  Future<List<ClinicalCaseModel>> checkSimilarCases(String userPrompt) async {
    final userId = userService.currentUser.value.id;
    return await clinicalCaseServive.detectSimilarCases(userId, userPrompt);
  }

  Future<void> showInteractiveSummaryIfAvailable() async {
    // Función placeholder - implementar lógica específica si es necesaria
    print('showInteractiveSummaryIfAvailable called');
    if (hasHiddenInteractiveSummary) {
      final summary = takeInteractiveSummary();
      if (summary != null) {
        interactiveEvaluationGenerated.value = true;
      }
    }
  }
}
