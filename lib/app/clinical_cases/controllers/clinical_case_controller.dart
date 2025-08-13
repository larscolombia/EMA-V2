import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/life_stage.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/sex_and_status.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/services/clinical_cases_services.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';
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
  final clinicalCaseServive = Get.find<ClinicalCasesServices>();
  final uiObserverService = Get.find<UiObserverService>();
  final userService = Get.find<UserService>();
  final profileController = Get.find<ProfileController>();

  final Rx<ClinicalCaseModel?> currentCase = Rx(null);
  final messages = <ChatMessageModel>[].obs;
  final questions = <QuestionResponseModel>[].obs;

  final Rx<QuestionResponseModel?> currentQuestion = Rx(null);

  final isComplete = false.obs;
  final isTyping = false.obs; // New observable

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
  }) async {
    // Prevent multiple simultaneous calls
    if (isTyping.value) {
      print(
        '⚠️ [ClinicalCaseController] generateCase already in progress, ignoring duplicate call',
      );
      return;
    }

    try {
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

      _startClinicalCase(temporalCase);

      // Descontar la cuota después de crear el caso clínico
      final success = await profileController.decrementClinicalCaseQuota();
      if (success) {
        profileController.refreshClinicalCaseQuota();
      }
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

  Future<void> _startClinicalCase(ClinicalCaseModel temporalCase) async {
    try {
      final generated = await clinicalCaseServive.generateCase(temporalCase);
      final clinicalCase = generated.clinicalCase;

      Logger.mini('_startClinicalCase');
      Logger.objectValue('clinicalCase.textPlane', clinicalCase.textPlane);

      if (clinicalCase.type == ClinicalCaseType.analytical) {
        final newMessages = await clinicalCaseServive.startAnalytical(
          clinicalCase,
        );
        _insertMessages(newMessages);
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
        isTyping.value = false; // Make sure to set typing to false
        return;
      } else {
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

      final aiMessage = await clinicalCaseServive.sendMessage(
        userMessage,
      ); // Await AI response
      messages.add(aiMessage);

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
}
