import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_generate_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';
import 'package:ema_educacion_medica_avanzada/core/logger/logger.dart';


class ApiClinicalCaseFake {
  Future<ClinicalCaseGenerateData> generateCase(ClinicalCaseModel clinicalCase) async {
    try {
      final response = {
          "uuid": clinicalCase.uid,
          "success": true,
          "case": {
              "id": DateTime.now().minute.toString() + (DateTime.now().second * 10).toString(),
              "title": "",
              "type": "static",
              "age": "adolescente",
              "sex": "male",
              "gestante": 0,
              "anamnesis": "Paciente masculino de 15 años que acude a la consulta por presentar dolor abdominal agudo localizado en la fosa ilíaca derecha desde hace 12 horas. Refiere náuseas y fiebre de 38.5°C. Niega haber tenido síntomas similares en el pasado. No hay antecedentes personales relevantes ni alergias conocidas.",
              "physical_examination": "El examen físico revela un abdomen blando pero con dolor localizado en la fosa ilíaca derecha. Signo de rebote positivo. El paciente presenta leve taquicardia, pero no hay hallazgos pulmonares o cardíacos significativos.",
              "diagnostic_tests": "El análisis sanguíneo muestra leucocitosis con predominio de neutrófilos. Una ecografía abdominal revela signos compatibles con apendicitis aguda.",
              "final_diagnosis": "Apendicitis aguda.",
              "management": "Se decide realizar apendicectomía de emergencia. El procedimiento se lleva a cabo sin complicaciones y el paciente es ingresado a recuperación postquirúrgica. Se administra terapia antibiótica profiláctica con ceftriaxona. El paciente evoluciona favorablemente y es dado de alta a los dos días con recomendaciones postoperatorias y cita de seguimiento.",
              "created_at": "2025-02-25 02:58:33",
              "updated_at": "2025-02-25 02:58:33"
          },
          "thread_id": "thread_jCrpVE4WqXo97VzM6HOBup22"
      };

      final generateClinicalCase = ClinicalCaseModel.fromApi(response);

      final completeClinicalCase = generateClinicalCase.copyWith(
        type: clinicalCase.type,
        uid: clinicalCase.uid,
        userId: clinicalCase.userId,
      );

      return ClinicalCaseGenerateData(
        clinicalCase: completeClinicalCase,
        question: null,
      );
    } catch (e) {
      Logger.error(e.toString());
      throw Exception('No fue posible crear el caso clínico.');
    }
  }
  
  Future<QuestionResponseModel> sendAnswerMessage(QuestionResponseModel questionWithAnswer) async {
    return questionWithAnswer.copyWith(
      fit: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit.',
    );
  }

  Future<ChatMessageModel> sendMessage(ChatMessageModel userMessage) async {
    return ChatMessageModel.ai(
      chatId: userMessage.chatId,
      text: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, pero más extenso, capicci.',
    );
  }
  
  Future<List<ClinicalCaseModel>> getClinicalCaseByUserId({required String userId}) async {
    await Future.delayed(Duration(seconds: 1));
    
    List<ClinicalCaseModel> quizzes = [];
    return quizzes;
  }
}
