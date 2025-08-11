// https://www.sqlitetutorial.net/sqlite-where/
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/interfaces/clinical_case_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';


class LocalClinicalCaseData extends ILocalData<ClinicalCaseModel> implements IClinicalCaseLocalData {
  static final String _tableName = 'clinical_cases_v1';
  @override String get tableName => _tableName;
  @override String get singular => 'el caso clínico';
  @override String get plural => 'los casos clínicos';
  
  Future<List<QuizModel>> getByCategoryId(int categoryId, [int page = 1, int limit = 25]) async {
    await Future.delayed(Duration(seconds: 2));
    
    List<QuizModel> quizzes = [];

    return quizzes;
  }
 
  @override
  ClinicalCaseModel fromApi(Map<String, dynamic> map) {
    return ClinicalCaseModel.fromApi(map);
  }
  
  @override
  ClinicalCaseModel fromMap(Map<String, dynamic> map) {
    return ClinicalCaseModel.fromMap(map);
  }
  
  @override
  Map<String, dynamic> toMap(item) {
    final quiz = item as ClinicalCaseModel;
    return quiz.toMap();
  }

  static String sqlInstructionCreateTable() {
    return '''
      CREATE TABLE $_tableName (
        uid TEXT PRIMARY KEY,
        remoteId INTEGER NOT NULL,
        threadId TEXT NOT NULL,
        userId INTEGER NOT NULL,
        title TEXT NOT NULL,
        type TEXT NOT NULL,
        age TEXT NOT NULL,
        sex TEXT NOT NULL,
        gestante INTEGER NOT NULL,
        isReal INTEGER NOT NULL,
        anamnesis TEXT,
        physicalExamination TEXT,
        diagnosticTests TEXT,
        finalDiagnosis TEXT,
        management TEXT,
        createdAt INTEGER NOT NULL,
        updatedAt INTEGER NOT NULL,
        feedback TEXT
      );
    ''';
  }
}
