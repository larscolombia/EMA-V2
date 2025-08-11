import 'clinical_case_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';

class ClinicalCaseGenerateData {
  final ClinicalCaseModel clinicalCase;
  final QuestionResponseModel? question;

  ClinicalCaseGenerateData({
    required this.clinicalCase,
    this.question,
  });

  ClinicalCaseGenerateData copyWith({
    ClinicalCaseModel? clinicalCase,
    QuestionResponseModel? question,
  }) {
    return ClinicalCaseGenerateData(
      clinicalCase: clinicalCase ?? this.clinicalCase,
      question: question ?? this.question,
    );
  }
}
