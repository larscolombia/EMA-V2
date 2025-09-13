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

  /// Obtiene los N casos más recientes con sus resúmenes para detección de similitud
  Future<List<ClinicalCaseModel>> getRecentCasesForSimilarity(int userId, {int limit = 20}) async {
    final where = 'userId = ? AND summary IS NOT NULL AND summary != ""';
    final whereArgs = [userId];
    
    final items = await getItems(
      where: where,
      whereArgs: whereArgs,
      orderBy: 'createdAt DESC',
      limit: limit,
    );
    
    return items;
  }

  /// Busca casos que contengan palabras clave similares en el resumen
  Future<List<ClinicalCaseModel>> findSimilarCases(
    int userId, 
    String searchText, 
    {int limit = 10}
  ) async {
    // Extraer palabras clave del texto de búsqueda
    final keywords = searchText
        .toLowerCase()
        .replaceAll(RegExp(r'[^\w\s]'), '')
        .split(' ')
        .where((word) => word.length > 3)
        .take(5)
        .toList();
    
    if (keywords.isEmpty) return [];
    
    // Crear consulta LIKE para buscar cualquiera de las palabras clave
    final likeConditions = keywords.map((_) => 'summary LIKE ?').join(' OR ');
    final where = 'userId = ? AND summary IS NOT NULL AND ($likeConditions)';
    final whereArgs = [userId, ...keywords.map((k) => '%$k%')];
    
    final items = await getItems(
      where: where,
      whereArgs: whereArgs,
      orderBy: 'createdAt DESC',
      limit: limit,
    );
    
    return items;
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
        feedback TEXT,
        summary TEXT
      );
    ''';
  }
}
