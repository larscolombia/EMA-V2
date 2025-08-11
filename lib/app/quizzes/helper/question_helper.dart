// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';

class QuestionHelper {
  static List<String>getOptions(Map<String, dynamic> map) {
    print('DEBUG: getOptions called with map: $map');
    
    final options = map['options'] ?? map['opciones'];
    print('DEBUG: raw options value: $options');
    
    if (options == null) {
      print('DEBUG: options is null, returning empty list');
      return [];
    }

    try {
      final dynamic decoded = (options is String) ? jsonDecode(options) : options;
      print('DEBUG: decoded options: $decoded');

      if (decoded is List) {
        final result = decoded.map((s) => s.toString()).toList();
        print('DEBUG: returning list result: $result');
        return result;
      }

      if (options is List) {
        final result = options.map((s) => s.toString()).toList();
        print('DEBUG: returning options list result: $result');
        return result;
      }
      
      print('DEBUG: no valid list found, returning empty list');
      return [];
    } catch (e) {
      print('DEBUG: error processing options: $e');
      // Si falla el jsonDecode o la conversión, retornamos null
      return [];
    }
  }

  static Future<QuestionResponseModel> questionWithDelay(QuestionResponseModel question, bool withoutDelay) async {
    final Duration delay = question.isAnswered == true || withoutDelay
      ? const Duration(seconds: 0)
      : const Duration(seconds: 3);

      return await Future.delayed(delay, () => question);
  }
}
