import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';

void main() {
  print('Testing clinical case JSON processing...');
  
  try {
    // Crear datos de prueba para simular un caso clínico
    final testQuestionMap = {
      'id': 1,
      'question': '¿Cuál es el diagnóstico más probable?',
      'type': 'single_choice',
      'options': ['Opción A', 'Opción B', 'Opción C', 'Opción D'],
      'answer': 'Opción A',
    };
    
    final testQuestion = QuestionResponseModel.fromClinicalCaseApi(
      quizId: 'test-clinical-case-1',
      feedback: 'Excelente respuesta',
      questionMap: testQuestionMap,
    );
    
    print('Test successful!');
    print('Question: ${testQuestion.question}');
    print('Type: ${testQuestion.type}');
    print('Options: ${testQuestion.options}');
    print('Answer: ${testQuestion.answer}');
    print('Parent Type: ${testQuestion.parentType}');
  } catch (e) {
    print('Test failed with error: $e');
  }
} 