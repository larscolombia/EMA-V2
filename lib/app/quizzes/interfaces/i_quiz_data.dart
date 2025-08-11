import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/quiz_generate_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/quiz_model.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';


abstract class IQuizRemoteData {
  // Commands
  Future<QuizGenerateData> generateQuizData(QuizModel quiz);
  Future<QuizModel> evaluateQuiz(QuizModel quiz);

  // Queries
  Future<QuizModel> getQuizById(String quizId);
  Future<List<QuestionResponseModel>> getQuestions(QuizModel quiz);
}

abstract class IQuestionsLocalData implements ILocalData<QuestionResponseModel>  {}

abstract class IQuizzLocalData implements ILocalData<QuizModel>  {}
