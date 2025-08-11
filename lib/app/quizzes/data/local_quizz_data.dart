// https://www.sqlitetutorial.net/sqlite-where/
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';


class LocalQuizzData extends ILocalData<QuizModel> implements IQuizzLocalData {
  static final String _tableName = 'quizzes_v1';
  @override String get tableName => _tableName;
  @override String get singular => 'el cuestionario';
  @override String get plural => 'los cuestionarios';
  
  Future<List<QuizModel>> getByCategoryId(int categoryId, [int page = 1, int limit = 25]) async {
    await Future.delayed(Duration(seconds: 2));
    
    List<QuizModel> quizzes = [];

    return quizzes;
  }
 
  @override
  QuizModel fromApi(Map<String, dynamic> map) {
    return QuizModel.fromApi(map);
  }
  
  @override
  QuizModel fromMap(Map<String, dynamic> map) {
    return QuizModel.fromMap(map);
  }
  
  @override
  Map<String, dynamic> toMap(item) {
    final quiz = item as QuizModel;
    return quiz.toMap();
  }

  static String sqlInstructionCreateTable() {
    return '''
      CREATE TABLE $_tableName (
        uid TEXT PRIMARY KEY NOT NULL,
        remoteId INTEGER,
        userId INTEGER NOT NULL,
        shortTitle TEXT NOT NULL,
        title TEXT NOT NULL,
        numQuestions INTEGER NOT NULL,
        numAnswers INTEGER NOT NULL,
        score INTEGER,
        feedback TEXT NOT NULL,
        level INTEGER NOT NULL,
        animated INTEGER NOT NULL,
        createdAt INTEGER NOT NULL,
        updatedAt INTEGER NOT NULL,
        categoryId INTEGER
      );
    ''';
  }
}
