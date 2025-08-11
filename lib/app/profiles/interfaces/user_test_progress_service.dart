import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/models/most_studied_category.dart';

abstract class UserTestProgressService {
  Future<TestProgressData> fetchTestScores({
    required int userId,
    required String authToken,
  });

  Future<List<MonthlyScore>> fetchMonthlyScores({
    required int userId,
    required String authToken,
  });

  // Nuevo método para la versión premium
  Future<MostStudiedCategory> fetchMostStudiedCategory({
    required int userId,
    required String authToken,
  });

  // Nuevos métodos
  Future<int> fetchTotalTests({
    required int userId,
    required String authToken,
  });

  Future<int> fetchTotalChats({
    required int userId,
    required String authToken,
  });

  // Nuevo método para casos clínicos
  Future<int> fetchClinicalCasesCount({
    required int userId,
    required String authToken,
  });
}
