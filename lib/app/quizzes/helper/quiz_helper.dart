import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';


class QuizHelper {
  static String title(String? categoryName, int numQuestions, QuizzLevel level) {
    String title = 'Este cuestionario evalúa conocimientos de ${categoryName?.toLowerCase() ?? 'medicina general'}.';
    String levelDescription = level == QuizzLevel.hard
      ? '$numQuestions preguntas de nivel ${level.name.toLowerCase()}.'
      : '$numQuestions preguntas.';

    return '$title $levelDescription';
  }

  static String shortTitle(String? categoryName, int numQuestions, QuizzLevel level) {
    // Este cuestionario evalúa conocimientos de medicina general, n pregutnas. // 
    return 'Cuestionario ${categoryName ?? 'Medicina General'}. ($numQuestions)';
  }
}
