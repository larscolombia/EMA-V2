import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/app/profiles/models/most_studied_category.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:http/http.dart' as http;

import '../../../config/constants/constants.dart';

class ApiUserTestProgressService extends UserTestProgressService {
  @override
  Future<TestProgressData> fetchTestScores({
    required int userId,
    required String authToken,
  }) async {
    final url = Uri.parse('$apiUrl/user/$userId/test-progress');
    final response = await http.get(
      url,
      headers: {
        'Authorization': 'Bearer $authToken',
        'Accept': 'application/json',
      },
    );

    if (response.statusCode == 200) {
      final jsonResponse = jsonDecode(response.body);
      final resumen = jsonResponse['data']['resumen'];
      // print(resumen);
      return TestProgressData.fromJson({
        'data': {'resumen': resumen}
      });
    } else if (response.statusCode == 404) {
      final jsonResponse = jsonDecode(response.body);
      throw Exception(jsonResponse['message'] ?? 'Usuario no encontrado.');
    } else {
      throw Exception(
          'Error al obtener los puntos de evaluación: ${response.statusCode}');
    }
  }

  @override
  Future<List<MonthlyScore>> fetchMonthlyScores({
    required int userId,
    required String authToken,
  }) async {
    final url = Uri.parse('$apiUrl/user/$userId/test-progress');
    final response = await http.get(
      url,
      headers: {
        'Authorization': 'Bearer $authToken',
        'Accept': 'application/json',
      },
    );

    if (response.statusCode == 200) {
      final jsonResponse = jsonDecode(response.body);
      final data = jsonResponse['data'];
      // Se extrae la lista de puntos mensuales del campo "puntos_meses"
      final List<dynamic> puntosMeses = data['puntos_meses'];
      return puntosMeses.map((item) => MonthlyScore.fromJson(item)).toList();
    } else if (response.statusCode == 404) {
      final jsonResponse = jsonDecode(response.body);
      throw Exception(jsonResponse['message'] ?? 'Usuario no encontrado.');
    } else {
      throw Exception(
          'Error al obtener los puntos mensuales: ${response.statusCode}');
    }
  }

  @override
  Future<MostStudiedCategory> fetchMostStudiedCategory({
    required int userId,
    required String authToken,
  }) async {
    final url = Uri.parse('$apiUrl/user/$userId/most-studied-category');
    final response = await http.get(
      url,
      headers: {
        'Authorization': 'Bearer $authToken',
        'Accept': 'application/json',
      },
    );

    if (response.statusCode == 200) {
      final jsonResponse = jsonDecode(response.body);
      final data = jsonResponse['data'];

      return MostStudiedCategory.fromJson(data);
    } else {
      final jsonResponse = jsonDecode(response.body);
      throw Exception(
          jsonResponse['message'] ?? 'Error fetching premium data.');
    }
  }

  @override
  Future<int> fetchTotalTests({
    required int userId,
    required String authToken,
  }) async {
    final url = Uri.parse('$apiUrl/user/$userId/total-tests');
    final response = await http.get(
      url,
      headers: {
        'Authorization': 'Bearer $authToken',
        'Accept': 'application/json',
      },
    );
    if (response.statusCode == 200) {
      final jsonResponse = jsonDecode(response.body);
      return jsonResponse['data']['total_tests'] as int;
    } else {
      throw Exception(
          'Error al obtener total de cuestionarios: ${response.statusCode}');
    }
  }

  @override
  Future<int> fetchTotalChats({
    required int userId,
    required String authToken,
  }) async {
    final url = Uri.parse('$apiUrl/chats/$userId');
    final response = await http.get(
      url,
      headers: {
        'Authorization': 'Bearer $authToken',
        'Accept': 'application/json',
      },
    );
    if (response.statusCode == 200) {
      final jsonResponse = jsonDecode(response.body);
      return int.parse(jsonResponse['total_chats'].toString());
    } else {
      throw Exception(
          'Error al obtener total de chats: ${response.statusCode}');
    }
  }

  @override
  Future<int> fetchClinicalCasesCount({
    required int userId,
    required String authToken,
  }) async {
    final url = Uri.parse('$apiUrl/users/$userId/clinical-cases-count');
    final response = await http.get(
      url,
      headers: {
        'Authorization': 'Bearer $authToken',
        'Accept': 'application/json',
      },
    );
    if (response.statusCode == 200) {
      final jsonResponse = jsonDecode(response.body);
      return jsonResponse['clinical_cases_count'] as int;
    } else {
      throw Exception(
          'Error al obtener casos clínicos: ${response.statusCode}');
    }
  }
}
