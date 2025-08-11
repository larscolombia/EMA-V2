import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/test_score.dart';
import '../models/monthly_score.dart';
import '../models/most_studied_category.dart';

class StatisticsRepository {
  static const _keyTestScores = 'cached_test_scores';
  static const _keyMonthlyScores = 'cached_monthly_scores';
  static const _keyMostStudiedCategory = 'cached_most_studied_category';
  static const _keyTotalTests = 'cached_total_tests';
  static const _keyTotalChats = 'cached_total_chats';
  static const _keyClinicalCasesCount = 'cached_clinical_cases_count';
  static const _keyCacheTimestamp = 'cached_timestamp';

  // Actualiza el método para guardar también totalTests y totalChats
  Future<void> cacheStatistics({
    required List<TestScore> testScores,
    required List<MonthlyScore> monthlyScores,
    MostStudiedCategory? mostStudiedCategory,
    required int totalTests,
    required int totalChats,
    required int clinicalCasesCount,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    prefs.setString(
        _keyTestScores,
        jsonEncode(testScores
            .map((e) => {
                  'test_id': e.testId,
                  'test_name': e.testName,
                  'score_obtained': e.scoreObtained,
                  'max_score': e.maxScore,
                })
            .toList()));
    prefs.setString(
        _keyMonthlyScores,
        jsonEncode(monthlyScores
            .map((e) => {
                  'mes': e.mes,
                  'puntos': e.puntos,
                })
            .toList()));
    if (mostStudiedCategory != null) {
      prefs.setString(
          _keyMostStudiedCategory,
          jsonEncode({
            'category_id': mostStudiedCategory.categoryId,
            'category_name': mostStudiedCategory.categoryName,
          }));
    }
    prefs.setInt(_keyTotalTests, totalTests);
    prefs.setInt(_keyTotalChats, totalChats);
    prefs.setInt(_keyClinicalCasesCount, clinicalCasesCount);
    prefs.setInt(_keyCacheTimestamp, DateTime.now().millisecondsSinceEpoch);
  }

  // Recupera la caché, incluyendo los nuevos datos si existen
  Future<Map<String, dynamic>?> getCachedStatistics() async {
    final prefs = await SharedPreferences.getInstance();
    final timestamp = prefs.getInt(_keyCacheTimestamp);
    if (timestamp == null ||
        DateTime.now()
                .difference(DateTime.fromMillisecondsSinceEpoch(timestamp))
                .inHours >=
            1) {
      return null;
    }
    final testScoresStr = prefs.getString(_keyTestScores);
    final monthlyScoresStr = prefs.getString(_keyMonthlyScores);
    final mostStudiedCategoryStr = prefs.getString(_keyMostStudiedCategory);
    final totalTests = prefs.getInt(_keyTotalTests);
    final totalChats = prefs.getInt(_keyTotalChats);
    final clinicalCasesCount = prefs.getInt(_keyClinicalCasesCount);
    if (testScoresStr == null || monthlyScoresStr == null) return null;
    final testScoresJson = jsonDecode(testScoresStr);
    final testScores = (testScoresJson as List)
        .map((e) => TestScore.fromJson(e as Map<String, dynamic>))
        .toList();
    final monthlyScoresJson = jsonDecode(monthlyScoresStr);
    final monthlyScores = (monthlyScoresJson as List)
        .map((e) => MonthlyScore.fromJson(e as Map<String, dynamic>))
        .toList();
    MostStudiedCategory? mostStudiedCategory;
    if (mostStudiedCategoryStr != null) {
      final categoryJson = jsonDecode(mostStudiedCategoryStr);
      mostStudiedCategory =
          MostStudiedCategory.fromJson(categoryJson as Map<String, dynamic>);
    }
    return {
      'testScores': testScores,
      'monthlyScores': monthlyScores,
      'mostStudiedCategory': mostStudiedCategory,
      'totalTests': totalTests ?? 0,
      'totalChats': totalChats ?? 0,
      'clinicalCasesCount': clinicalCasesCount ?? 0,
    };
  }

  // Nuevo método para borrar la caché al cerrar sesión
  Future<void> clearCache() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keyTestScores);
    await prefs.remove(_keyMonthlyScores);
    await prefs.remove(_keyMostStudiedCategory);
    await prefs.remove(_keyCacheTimestamp);
  }
}
