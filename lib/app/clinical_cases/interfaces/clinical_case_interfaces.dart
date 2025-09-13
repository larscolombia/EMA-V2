
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_model.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';


abstract class IClinicalCaseLocalData implements ILocalData<ClinicalCaseModel> {
  /// Obtiene los N casos más recientes con sus resúmenes para detección de similitud
  Future<List<ClinicalCaseModel>> getRecentCasesForSimilarity(int userId, {int limit = 20});
  
  /// Busca casos que contengan palabras clave similares en el resumen
  Future<List<ClinicalCaseModel>> findSimilarCases(int userId, String searchText, {int limit = 10});
}

abstract class IClinicalCaseMessageLocalData implements ILocalData<ChatMessageModel>  {}
