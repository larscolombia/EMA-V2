// ignore_for_file: public_member_api_docs, sort_constructors_first
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/quiz_generate_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';


class ApiQuizzData implements IQuizRemoteData {
  final Dio _dio = Get.find<ApiService>().dio;

  @override
  Future<QuizModel> evaluateQuiz(QuizModel quiz) async {
    try {
      final quizToEvaluate = quiz.toEvaluationMap();

      final url = '/tests/responder-test/submit';

      final response = await _dio.post(url, data: quizToEvaluate);

      if (response.statusCode == 200) {
        final data = await response.data;

        final evaluation = _getEvaluation(data['evaluation']);

        final questions = _updateQuestions(quiz.questions, evaluation);

        final score = data['correct_answers'] ?? 0;

        final feedback = data['fit_global'] ?? '';
        
        return quiz.copyWith(score: score, feedback: feedback, questions: questions);
      }

      throw Exception('Error ${response.statusCode} al evaluar quiz.');
      
    } catch (e) {
      Logger.error(e.toString(), className: 'ApiQuizzData', methodName: 'evaluateQuiz', meta: 'quizId: ${quiz.uid}');
      throw Exception('Error: ${e.toString()}');
    }
  }

  @override
  Future<QuizGenerateData> generateQuizData(QuizModel quiz) async {
    try {
      final body = quiz.toRequestBody();
      final url = '/tests/generate/${quiz.userId}';

      print('DEBUG: URL: $url');
      print('DEBUG: Request body: $body');

      final response = await _dio.post(url, data: body);

      print('DEBUG: Response status: ${response.statusCode}');
      print('DEBUG: Response data: ${response.data}');

      final generateQuiz = QuizGenerateData.fromApi(response.data, quiz.uid, quiz.userId);

      return generateQuiz;
    } catch (e) {
      print('DEBUG: Error in generateQuizData: $e');
      Logger.error(e.toString());
      throw Exception('No fue posible crear el cuestionario.');
    }
  }

  @override
  Future<List<QuestionResponseModel>> getQuestions(QuizModel quiz) async {
    try {

      dynamic response;
      List<QuestionResponseModel> questions = [];

      response = await _dio.get('/tests/${quiz.uid}/questions');

      if (response.statusCode == 200) {
        final resBody = await response.stream.bytesToString();

        final array = jsonDecode(resBody);

        questions = array.map((q) => QuestionResponseModel.fromApi(q)).toList();
      }

      return questions;
    } catch (e) {
      throw Exception('Error al obtener respuestas');
    }
  }
  
  @override
  Future<QuizModel> getQuizById(String quizId) async {
    try {

      dynamic response;

      response = await _dio.get('/user/{{user_id}}/test/$quizId/details'); 

      if (response.statusCode == 200) {
        final resBody = await response.stream.bytesToString();

        final data = jsonDecode(resBody);

        final quiz = QuizModel.fromApi(data); // Adaptar el endpoint al método fromApi

        return quiz;
      }

      throw Exception('Error ${response.statusCode} al obtener el cuestarionario.');
    } catch (e) {
      throw Exception('Error al obtener respuestas');
    }
  }

  List<QuestionResponseModel> _updateQuestions(List<QuestionResponseModel> oldQuestions, List<Map<String, dynamic>> evaluations) {
    final updatedQuestions = oldQuestions.map((question) {
      final evaluation = evaluations.firstWhere((e) => e['question_id'] == question.remoteId);
      final isCorrect = evaluation['is_correct'] as int == 1;
      final fit = evaluation['fit'] != null ? evaluation['fit'] as String : '';

      return question.copyWith(isCorrect: isCorrect, fit: fit);
    }).toList();

    return updatedQuestions;
  }

  List<Map<String, dynamic>> _getEvaluation(dynamic data) {
    // final data = this['evaluation'];
    if (data is List) {
      return data.map((item) {
        if (item is Map<String, dynamic>) {
          return item;
        } else {
          throw FormatException('Invalid data format: expected Map<String, dynamic>');
        }
      }).toList();
    } else {
      throw FormatException('Invalid data: expected a List');
    }
  }
}
