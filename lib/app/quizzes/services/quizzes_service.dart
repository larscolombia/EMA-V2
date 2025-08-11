import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/quiz_generate_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:get/get.dart';


class QuizzesService extends GetxService {
  final _localQuizzesData = Get.find<LocalQuizzData>();
  final _localQuestionsData = Get.find<LocalQuestionsData>();
  final _quizRemoteData = Get.find<IQuizRemoteData>();

  // Commands
  Future<QuizGenerateData> generateQuiz(QuizModel quiz) async {
    // Obtenemos la información del endpoint que devuelve
    final generatedData = await _quizRemoteData.generateQuizData(quiz);

    // Actualizamos el quiz recibido desde el controlador
    final quizUpdated =  quiz.copyWith(remoteId: generatedData.testId);

    // Anexamos el quiz a la respuesta
    final updateGeneratedQuiz = generatedData.copyWithQuiz(quizUpdated);
    
    // Almacenamos en local el quiz y las preguntas
    await _localQuizzesData.insertOne(quizUpdated);
    await _localQuestionsData.insertMany(generatedData.questions);

    // Devolvemos la respuesta que contiene el quiz y las preguntas
    return updateGeneratedQuiz;
  }
  
  Future<QuizModel> evaluateQuiz(QuizModel quiz) async {
    final evaluatedQuiz = await _quizRemoteData.evaluateQuiz(quiz);

    await _localQuizzesData.update(evaluatedQuiz, 'uid = ?', [quiz.uid]);

    final where = 'remoteId = ? AND quizId = ?';

    for (var q in evaluatedQuiz.questions) {
      final whereArgs = [q.remoteId, quiz.uid];
      await _localQuestionsData.update(q, where, whereArgs);
    }

    return evaluatedQuiz;
  }

  void markAsAnimated(QuizModel quiz) {
    String where = 'uid = ?';
    List<Object> whereArgs = [quiz.uid];

    final markAsAnimated = quiz.copyWith(animated: true);

    _localQuizzesData.update(markAsAnimated, where, whereArgs);
  }
  
  Future<QuestionResponseModel> saveAnswer(QuestionResponseModel question, AnswerModel userAnswer) async {
    final questionAnswered = question.copyWith(userAnswer: userAnswer.answer, userAnswers: userAnswer.answers);

    final where = 'id = ?';
    final whereArgs = [question.id];

    await _localQuestionsData.update(questionAnswered, where, whereArgs);

    return questionAnswered;
  }

  Future<QuizModel> updateQuiz(QuizModel quiz) async {
    await _localQuizzesData.update(quiz, 'uid = ?', [quiz.uid]);

    return quiz;
  }
  
  // Queries
  Future<QuizModel> getQuizById(String quizId) async {
    final where = 'uid = ?';
    final whereArgs = [quizId];

    final localQuiz = await _localQuizzesData.getById(where, whereArgs);

    if (localQuiz != null) {
      return localQuiz;
    }

    final remoteQuiz = await _quizRemoteData.getQuizById(quizId);

    await _localQuizzesData.insertOne(remoteQuiz);

    return remoteQuiz;
  }

  Future<List<QuestionResponseModel>> getQuestions(QuizModel quiz) async {
    final where = 'quizId = ?';
    final whereArgs = [quiz.uid];

    final localQuestions = await _localQuestionsData.getItems(where: where, whereArgs: whereArgs);

    if (localQuestions.isNotEmpty) {
      return localQuestions;
    }

    // Todo: implementar la funcionalidad remota, solicita el endpoint
    final remoteQuestions = await _quizRemoteData.getQuestions(quiz);

    await _localQuestionsData.insertMany(remoteQuestions);

    return remoteQuestions;
  }
}
