import 'package:ema_educacion_medica_avanzada/app/actions/models/action_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/services/actions_service.dart';
import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/quiz_generate_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/quiz_status.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/notify/notify.dart';
import 'package:ema_educacion_medica_avanzada/core/ui/ui_observer_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../profiles/profiles.dart';


class QuizController extends GetxController with StateMixin<QuizModel> {
  ScrollController? _quizScrollController;
  final _actionsService = Get.find<ActionsService>();
  final _quizzesService = Get.find<QuizzesService>();
  final _keyboardService = Get.find<UiObserverService>();
  final _userService = Get.find<UserService>();
  final profileController = Get.find<ProfileController>();

  final _index = 0.obs;

  final currentQuestion = QuestionResponseModel.empty().obs;
  final progress = 0.0.obs;
  final totalAnswers = 0.obs;
  final totalQuestions = 0.obs;
  final isComplete = false.obs;
  final isEvaluated = false.obs;

  final textLoading = 'Preparando Cuestionario...'.obs;

  final inChatList = <QuestionResponseModel>[].obs;
  final rxQuestions = <QuestionResponseModel>[].obs;
  final isTyping = false.obs;

  @override
  void onInit() {
    super.onInit();
    _keyboardService.isKeyboardVisible.listen((value) {
      if (value) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          scrollToBottom();
        });
      }
    });
  }

  Future<void> generate({
    required int numQuestions,
    required QuizzLevel level,
    CategoryModel? category
  }) async {
    try {
      if (!profileController.canCreateMoreQuizzes()) {
        Get.snackbar(
          'Límite alcanzado',
          'Has alcanzado el límite de cuestionarios en tu plan actual. Actualiza tu plan para crear más cuestionarios.',
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

      currentQuestion.value = QuestionResponseModel.empty();
      _index.value = 0;
      await _updateQuestions([]);

      change(null, status: RxStatus.loading());

      final generateQuiz = QuizModel.generate(
        numQuestions: numQuestions,
        userId: _userService.currentUser.value.id,
        level: level,
        categoryName: category?.name,
        categoryId: category?.id,
      );

      showDetail(quiz: generateQuiz);

      final QuizGenerateData generatedQuizData = await _quizzesService.generateQuiz(generateQuiz);

      final QuizModel quizUpdated = generatedQuizData.quiz!;

      final action = ActionModel.quizzes(
        quizUpdated.userId,
        quizUpdated.uid,
        quizUpdated.shortTitle,
        quizUpdated.title,
        quizUpdated.categoryId
      );

      _actionsService.insertAction(action);

      change(quizUpdated, status: RxStatus.loadingMore());

      _updateQuestions(generatedQuizData.questions);

  // Quota consumption centralized in backend (quiz_generate flow)
    } catch (e) {
      change(null, status: RxStatus.error(e.toString()));
    }
  }

  Future<void> saveAnswer({required QuestionResponseModel question, required AnswerModel answer}) async {
    try {
      isTyping.value = true; // Mostrar animación al iniciar
      rxQuestions[_index.value] = await _quizzesService.saveAnswer(question, answer);

      _updateAnswersAndProgress();

      final numAnswers = value!.numAnswers + 1;

      final updatedQuiz = value!.copyWith(numAnswers: numAnswers);

      await _quizzesService.updateQuiz(updatedQuiz);

      change(updatedQuiz, status: RxStatus.success());

      _selectNextQuestion();

      scrollToBottom();
    } catch (e) {
      Notify.snackbar('Cuestionarios', e.toString(), NotifyType.error);
    } finally {
      isTyping.value = false;
    }
  }

  Future<void> evaluateCurrentQuiz() async {
    try {
      final QuizModel? quiz = value;

      if (quiz == null) {
        throw Exception('No hay un quiz activo para evaluar.');
      }

      if (rxQuestions.isEmpty) {
        throw Exception('No hay preguntas para evaluar');
      }

      final quizToEvaluate = quiz.copyWith(questions: rxQuestions);

      textLoading.value = 'Evaluando Cuestionario...';

      change(quiz, status: RxStatus.loading());

      final quizEvaluated = await _quizzesService.evaluateQuiz(quizToEvaluate);

      showDetail(quiz: quizEvaluated);

      _quizzesService.markAsAnimated(quizEvaluated);
    } catch (e) {
      change(null, status: RxStatus.error(e.toString()));
    }
  }

  Future<void> useQuiz(String quizId) async {
    try {
      if (quizId != value?.uid) {
        rxQuestions.clear();
      }

      final quiz = await _quizzesService.getQuizById(quizId);

      change(quiz, status: RxStatus.loadingMore());

      _loadQuestionsForQuiz(quiz);

      showDetail(quiz: quiz);
    } catch (e) {
      // Todo: notificar error
      change(null, status: RxStatus.error(e.toString()));
    }
  }

  void showDetail({required QuizModel quiz}) {
    change(quiz, status: RxStatus.success());

    if (quiz.status == QuizStatus.reviewed) {
      Get.toNamed(Routes.quizFeedBack.path(quiz.uid));
    } else {
      Get.toNamed(Routes.quizDetail.path(quiz.uid));
    }
  }

  void setScrollController(ScrollController? scrollController) {
    _quizScrollController = scrollController;
  }

  Future<void> _updateQuestions(List<QuestionResponseModel> newQuestions) async {
    try {
      rxQuestions.value = newQuestions;
      _index.value = 0;
      currentQuestion.value =
          newQuestions.isNotEmpty ? newQuestions.first : QuestionResponseModel.empty();

      _updateNumQuestions();
      _updateAnswersAndProgress();
      _selectNextQuestion();
    } catch (e) {
      change(null, status: RxStatus.error(e.toString()));
    }
  }

  Future<void> _loadQuestionsForQuiz(QuizModel quiz) async {
    try {
      final loadedQuestions = await _quizzesService.getQuestions(quiz);

      rxQuestions
        ..clear()
        ..addAll(loadedQuestions);

      _index.value = 0;
      currentQuestion.value =
          loadedQuestions.isNotEmpty ? loadedQuestions.first : QuestionResponseModel.empty();

      if (rxQuestions.isEmpty) {
        Notify.snackbar(
            'Cuestionarios',
            'No se encontraron preguntas para el cuestionario.',
            NotifyType.info);
      } else {
        _updateNumQuestions();
        _updateAnswersAndProgress();
        _selectNextQuestion();
      }
    } catch (e) {
      change(null, status: RxStatus.error(e.toString()));
    }
  }

  void scrollToBottom() {
    if (_quizScrollController == null) return;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_quizScrollController == null) return;

      _quizScrollController?.animateTo(
        _quizScrollController!.position.maxScrollExtent,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeOut,
      );
    });
  }

  void _selectNextQuestion() {
    inChatList.value = rxQuestions.where((q) => q.isAnswered).toList();

    _index.value = rxQuestions.indexWhere((q) => !q.isAnswered);

    if (_index.value != -1) {
      currentQuestion.value = rxQuestions[_index.value];
    }
  }

  void _updateAnswersAndProgress() {
    int totalAnswered = 0;

    for (var q in rxQuestions) {
      if (q.isAnswered) {
        totalAnswered++;
      }
    }

    totalAnswers.value = totalAnswered;

    progress.value = totalQuestions.value == 0
      ? 0
      : (totalAnswered / totalQuestions.value);

    isComplete.value = totalQuestions.value > 0 && totalQuestions.value == totalAnswered;

    isEvaluated.value = value?.feedback.isNotEmpty ?? false;
  }

  void _updateNumQuestions() {
    totalQuestions.value = rxQuestions.length;
  }
}
