// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';

class QuizGenerateData {
  final int testId;
  final int userId;
  final String? threadId;
  List<QuestionResponseModel> questions;
  QuizModel? quiz;

  QuizGenerateData({
    required this.testId,
    required this.userId,
    this.threadId,
    this.questions = const [],
    this.quiz,
  });

  factory QuizGenerateData.fromApi(Map<String, dynamic> map, String quizId, int userId) {
    print('DEBUG: QuizGenerateData.fromApi - map: $map');
    print('DEBUG: QuizGenerateData.fromApi - quizId: $quizId');
    print('DEBUG: QuizGenerateData.fromApi - userId: $userId');
    
    // La respuesta tiene la estructura {success: true, data: {...}}
    final data = map['data'] as Map<String, dynamic>?;
    print('DEBUG: QuizGenerateData.fromApi - data: $data');
    
    if (data == null) {
      print('DEBUG: QuizGenerateData.fromApi - data is null!');
      throw Exception('Data is null in response');
    }
    
    final questionsData = data['questions'];
    print('DEBUG: QuizGenerateData.fromApi - questionsData: $questionsData');
    
    if (questionsData == null) {
      print('DEBUG: QuizGenerateData.fromApi - questionsData is null!');
      throw Exception('Questions data is null in response');
    }
    
    final mappedList = questionsData.map((x) {
      Map<String, dynamic> xx = x;
      xx['quizId'] = quizId;
      return xx;
    }).toList();
    
    print('DEBUG: QuizGenerateData.fromApi - mappedList: $mappedList');
    
    final questions = mappedList.map((q) => QuestionResponseModel.fromApi(q)).toList();
    
    print('DEBUG: QuizGenerateData.fromApi - questions: ${questions.length}');
    
    return QuizGenerateData(
      testId: data['test_id'] ?? 0,
      userId: userId,
      threadId: data['thread_id'] as String?,
      questions: List<QuestionResponseModel>.from(questions),
    );
  }

  QuizGenerateData copyWithQuiz(QuizModel quiz) {
    return QuizGenerateData(
      testId: testId,
      userId: userId,
      threadId: threadId,
      questions: questions,
      quiz: quiz,
    );
  }
}
