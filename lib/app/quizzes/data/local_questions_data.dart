// https://www.sqlitetutorial.net/sqlite-where/
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';


class LocalQuestionsData extends ILocalData<QuestionResponseModel> implements IQuestionsLocalData {
  static final String _tableName = 'questions_v1';
  @override String get tableName => _tableName;
  @override String get singular => 'la pregunta';
  @override String get plural => 'las preguntas';
  
  @override
  QuestionResponseModel fromApi(Map<String, dynamic> map) {
    return QuestionResponseModel.fromApi(map);
  }
  
  @override
  QuestionResponseModel fromMap(Map<String, dynamic> map) {
    return QuestionResponseModel.fromMap(map);
  }
  
  @override
  Map<String, dynamic> toMap(dynamic item) {
    final question = item as QuestionResponseModel;
    return question.toMap();
  }

  static String sqlInstructionCreateTable() {
    return '''
      CREATE TABLE $_tableName (
        id TEXT PRIMARY KEY,
        remoteId INTEGER NOT NULL,
        quizId TEXT NOT NULL,
        parentType TEXT NOT NULL, -- clinical_case|quiz|unknown
        message TEXT,
        question TEXT NOT NULL,
        answer TEXT,
        userAnswer TEXT,
        userAnswers TEXT NOT NULL, -- Almacenar como JSON
        type TEXT NOT NULL, -- open_ended|single_choice|true_false|unknown
        options TEXT NOT NULL, -- Almacenar como JSON
        isCorrect INTEGER,
        fit TEXT,
        createdAt INTEGER NOT NULL,
        updatedAt INTEGER NOT NULL
      );
    ''';
  }
}
