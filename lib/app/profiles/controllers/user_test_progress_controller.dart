import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'dart:convert';
import 'package:crypto/crypto.dart';
import '../repositories/statistics_repository.dart';
import '../models/most_studied_category.dart';

class UserTestProgressController extends GetxController {
  final UserTestProgressService progressService =
      Get.find<UserTestProgressService>();

  final RxList<TestScore> testScores = <TestScore>[].obs;
  final RxList<MonthlyScore> monthlyScores = <MonthlyScore>[].obs;

  final RxBool isLoadingTestScores = false.obs;
  final RxBool isLoadingMonthlyScores = false.obs;

  final Rx<MostStudiedCategory?> mostStudiedCategory =
      Rxn<MostStudiedCategory>();
  final RxBool isLoadingMostStudiedCategory = false.obs;

  final RxInt totalTests = 0.obs;
  final RxInt totalChats = 0.obs;
  final RxInt totalClinicalCases = 0.obs;

  final StatisticsRepository statisticsRepo = StatisticsRepository();

  // Campos actualizados para el nuevo sistema
  final RxInt totalScore = 0.obs; // Puntos totales obtenidos
  final RxInt totalMaxScore = 0.obs; // Puntos totales posibles
  final RxDouble averagePercentage = 0.0.obs; // Promedio general %

  Future<void> loadTestScores({
    required int userId,
    required String authToken,
  }) async {
    try {
      isLoadingTestScores.value = true;
      final progressData = await progressService.fetchTestScores(
        userId: userId,
        authToken: authToken,
      );
      testScores.assignAll(progressData.tests);
      totalTests.value = progressData.summary.totalTests;
      totalScore.value = progressData.summary.totalScore;
      totalMaxScore.value = progressData.summary.totalMaxScore;
      averagePercentage.value = progressData.summary.averagePercentage;
    } catch (e) {
      testScores.clear();
      totalTests.value = 0;
      totalScore.value = 0;
      totalMaxScore.value = 0;
      averagePercentage.value = 0.0;
    } finally {
      isLoadingTestScores.value = false;
    }
  }

  Future<void> loadMonthlyScores({
    required int userId,
    required String authToken,
  }) async {
    try {
      isLoadingMonthlyScores.value = true;
      final scores = await progressService.fetchMonthlyScores(
        userId: userId,
        authToken: authToken,
      );
      monthlyScores.assignAll(scores);
    } catch (e) {
      monthlyScores.clear();
    } finally {
      isLoadingMonthlyScores.value = false;
    }
  }

  Future<void> loadMostStudiedCategory({
    required int userId,
    required String authToken,
  }) async {
    try {
      isLoadingMostStudiedCategory.value = true;
      final category = await progressService.fetchMostStudiedCategory(
        userId: userId,
        authToken: authToken,
      );
      mostStudiedCategory.value = category;
    } catch (e) {
      mostStudiedCategory.value = null;
    } finally {
      isLoadingMostStudiedCategory.value = false;
    }
  }

  Future<void> loadTotalTests({
    required int userId,
    required String authToken,
  }) async {
    try {
      final tests = await progressService.fetchTotalTests(
        userId: userId,
        authToken: authToken,
      );
      totalTests.value = tests;
    } catch (e) {
      totalTests.value = 0;
    }
  }

  Future<void> loadTotalChats({
    required int userId,
    required String authToken,
  }) async {
    try {
      final chats = await progressService.fetchTotalChats(
        userId: userId,
        authToken: authToken,
      );
      totalChats.value = chats;
    } catch (e) {
      totalChats.value = 0;
    }
  }

  Future<void> loadClinicalCasesCount({
    required int userId,
    required String authToken,
  }) async {
    try {
      final count = await progressService.fetchClinicalCasesCount(
        userId: userId,
        authToken: authToken,
      );
      totalClinicalCases.value = count;
    } catch (e) {
      totalClinicalCases.value = 0;
    }
  }

  Future<void> refreshAllStatistics({
    required int userId,
    required String authToken,
  }) async {
    final cachedData = await statisticsRepo.getCachedStatistics();
    String? cachedHash;
    if (cachedData != null) {
      testScores.assignAll(cachedData['testScores']);
      monthlyScores.assignAll(cachedData['monthlyScores']);
      mostStudiedCategory.value = cachedData['mostStudiedCategory'];
      cachedHash = _computeStatisticsHash(
        cachedData['testScores'],
        cachedData['monthlyScores'],
        cachedData['mostStudiedCategory'],
      );
    }

    // fetchTestScores ya invalida el cache con forceRefresh: true
    // Los demás métodos usarán ese mismo cache actualizado
    await Future.wait([
      loadTestScores(userId: userId, authToken: authToken),
      loadMonthlyScores(userId: userId, authToken: authToken),
      loadMostStudiedCategory(userId: userId, authToken: authToken),
      loadTotalTests(userId: userId, authToken: authToken),
      loadTotalChats(userId: userId, authToken: authToken),
      loadClinicalCasesCount(userId: userId, authToken: authToken),
    ]);

    final newHash = _computeStatisticsHash(
      testScores,
      monthlyScores,
      mostStudiedCategory.value,
    );

    if (cachedHash == null || cachedHash != newHash) {
      await statisticsRepo.cacheStatistics(
        testScores: testScores,
        monthlyScores: monthlyScores,
        mostStudiedCategory: mostStudiedCategory.value,
        totalTests: totalTests.value,
        totalChats: totalChats.value,
        clinicalCasesCount: totalClinicalCases.value,
      );
    } else {
      if (cachedData != null) {
        testScores.assignAll(cachedData['testScores']);
        monthlyScores.assignAll(cachedData['monthlyScores']);
        mostStudiedCategory.value = cachedData['mostStudiedCategory'];
      }
    }
  }

  String _computeStatisticsHash(
    List testScoresList,
    List monthlyScoresList,
    dynamic category,
  ) {
    final testScoresJson = jsonEncode(
      testScoresList.map((e) => e.toString()).toList(),
    );
    final monthlyScoresJson = jsonEncode(
      monthlyScoresList.map((e) => e.toString()).toList(),
    );
    final categoryJson =
        category != null ? jsonEncode(category.toString()) : '';
    final combined = testScoresJson + monthlyScoresJson + categoryJson;
    return md5.convert(utf8.encode(combined)).toString();
  }

  /// Registra un test completado en el sistema de estadísticas
  Future<void> recordTestCompletion({
    required String authToken,
    required String testName,
    required int scoreObtained,
    required int maxScore,
    int? categoryId,
  }) async {
    try {
      print(
        '[STATS] Registrando test: $testName score=$scoreObtained/$maxScore categoryId=$categoryId',
      );

      await progressService.recordTestCompletion(
        authToken: authToken,
        testName: testName,
        scoreObtained: scoreObtained,
        maxScore: maxScore,
        categoryId: categoryId,
      );

      print('[STATS] Test registrado exitosamente, recargando estadísticas...');

      // Forzar recarga inmediata de estadísticas SIN cache
      final user = Get.find<UserService>().currentUser.value;
      if (user.id > 0) {
        // Invalidar cache del overview service
        progressService.invalidateCache();

        await Future.wait([
          loadTestScores(userId: user.id, authToken: authToken),
          loadMonthlyScores(userId: user.id, authToken: authToken),
          loadMostStudiedCategory(userId: user.id, authToken: authToken),
          loadTotalTests(userId: user.id, authToken: authToken),
          loadTotalChats(userId: user.id, authToken: authToken),
          loadClinicalCasesCount(userId: user.id, authToken: authToken),
        ]);
        print(
          '[STATS] Estadísticas recargadas: totalScore=${totalScore.value}/${totalMaxScore.value} totalTests=${totalTests.value} totalChats=${totalChats.value}',
        );
      }
    } catch (e) {
      // Registrar error pero no bloquear el flujo
      print('[ERROR] No se pudo registrar test: $e');
      rethrow;
    }
  }
}
