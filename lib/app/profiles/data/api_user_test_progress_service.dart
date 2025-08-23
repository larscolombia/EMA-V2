import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/app/profiles/models/most_studied_category.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:http/http.dart' as http;

import '../../../config/constants/constants.dart';

class ApiUserTestProgressService extends UserTestProgressService {
  Map<int, Map<String, dynamic>> _overviewCache = {}; // userId -> overview data

  Future<Map<String, dynamic>> _fetchOverview({required int userId, required String authToken, bool forceRefresh = false}) async {
    if (!forceRefresh && _overviewCache.containsKey(userId)) {
      return _overviewCache[userId]!;
    }
    final url = Uri.parse('$apiUrl/user-overview/$userId');
    final response = await http.get(url, headers: {
      'Authorization': 'Bearer $authToken',
      'Accept': 'application/json',
    });
    if (response.statusCode == 200) {
      final jsonResponse = jsonDecode(response.body);
      final data = jsonResponse['data'] as Map<String, dynamic>;
      try {
        final profile = data['profile'] as Map<String, dynamic>?;
        final active = profile?['active_subscription'] as Map<String, dynamic>?;
        if (active != null) {
          final plan = (active['subscription_plan'] as Map<String, dynamic>?)?['name'];
          final cons = active['consultations'];
          final quest = active['questionnaires'];
            final clin = active['clinical_cases'];
            final files = active['files'];
          // Debug print visible in Flutter console
          // ignore: avoid_print
          print('[QUOTA] plan=$plan consultations=$cons questionnaires=$quest clinical_cases=$clin files=$files');
        }
      } catch (_) {}
      _overviewCache[userId] = data; // cache entire overview
      return data;
    }
    throw Exception('Error overview: ${response.statusCode}');
  }
  @override
  Future<TestProgressData> fetchTestScores({
    required int userId,
    required String authToken,
  }) async {
  final data = await _fetchOverview(userId: userId, authToken: authToken);
  // Stubs currently empty; adapt when backend adds real fields
  final resumen = (data['stats']?['test_progress'] ?? []);
  return TestProgressData.fromJson({'data': {'resumen': resumen}});
  }

  @override
  Future<List<MonthlyScore>> fetchMonthlyScores({
    required int userId,
    required String authToken,
  }) async {
  final data = await _fetchOverview(userId: userId, authToken: authToken);
  final List<dynamic> puntosMeses = data['stats']?['test_progress'] ?? [];
  return puntosMeses.map((item) => MonthlyScore.fromJson(item)).toList();
  }

  @override
  Future<MostStudiedCategory> fetchMostStudiedCategory({
    required int userId,
    required String authToken,
  }) async {
  final data = await _fetchOverview(userId: userId, authToken: authToken);
  // Stubbed -> returns empty/placeholder
  return MostStudiedCategory.fromJson(data['stats'] ?? {});
  }

  @override
  Future<int> fetchTotalTests({
    required int userId,
    required String authToken,
  }) async {
  final data = await _fetchOverview(userId: userId, authToken: authToken);
  return (data['stats']?['total_tests'] ?? 0) as int;
  }

  @override
  Future<int> fetchTotalChats({
    required int userId,
    required String authToken,
  }) async {
  final data = await _fetchOverview(userId: userId, authToken: authToken);
  final chats = data['stats']?['chats'] as List<dynamic>?;
  return chats?.length ?? 0;
  }

  @override
  Future<int> fetchClinicalCasesCount({
    required int userId,
    required String authToken,
  }) async {
  final data = await _fetchOverview(userId: userId, authToken: authToken);
  return (data['stats']?['clinical_cases_count'] ?? 0) as int;
  }
}
