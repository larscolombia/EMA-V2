import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
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

  final RxInt totalPreguntas = 0.obs;
  final RxInt totalCorrectas = 0.obs;
  final RxInt totalIncorrectas = 0.obs;

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
      totalPreguntas.value = progressData.summary.totalPreguntas;
      totalCorrectas.value = progressData.summary.totalCorrectas;
      totalTests.value = progressData.summary.totalTests;
      totalIncorrectas.value = progressData.summary.totalIncorrectas;
    } catch (e) {
      testScores.clear();
      totalPreguntas.value = 0;
      totalCorrectas.value = 0;
      totalIncorrectas.value = 0;
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
          userId: userId, authToken: authToken);
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
          userId: userId, authToken: authToken);
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
          userId: userId, authToken: authToken);
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
          userId: userId, authToken: authToken);
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

    await Future.wait([
      loadTestScores(userId: userId, authToken: authToken),
      loadMonthlyScores(userId: userId, authToken: authToken),
      loadMostStudiedCategory(userId: userId, authToken: authToken),
      loadTotalTests(userId: userId, authToken: authToken),
      loadTotalChats(userId: userId, authToken: authToken),
      loadClinicalCasesCount(userId: userId, authToken: authToken),
    ]);

    final newHash = _computeStatisticsHash(
        testScores, monthlyScores, mostStudiedCategory.value);

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
    final testScoresJson =
        jsonEncode(testScoresList.map((e) => e.toString()).toList());
    final monthlyScoresJson =
        jsonEncode(monthlyScoresList.map((e) => e.toString()).toList());
    final categoryJson =
        category != null ? jsonEncode(category.toString()) : '';
    final combined = testScoresJson + monthlyScoresJson + categoryJson;
    return md5.convert(utf8.encode(combined)).toString();
  }
}
