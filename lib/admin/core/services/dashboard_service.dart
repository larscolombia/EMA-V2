import 'dart:convert';
import 'package:ema_educacion_medica_avanzada/admin/core/models/dashboard_stats.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/models/timeline_data.dart';
import 'package:ema_educacion_medica_avanzada/config/constants/constants.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;

class DashboardService {
  final _storage = const FlutterSecureStorage();

  Future<String?> _getToken() async {
    return await _storage.read(key: 'admin_auth_token');
  }

  Future<DashboardStats> getStats() async {
    try {
      final token = await _getToken();
      final response = await http.get(
        Uri.parse('$apiUrl/admin/stats'),
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        final Map<String, dynamic> jsonResponse = json.decode(response.body);
        return DashboardStats.fromJson(jsonResponse['data']);
      } else if (response.statusCode == 401 || response.statusCode == 403) {
        throw Exception('No autorizado: se requiere rol de super_admin');
      } else {
        throw Exception(
          'Error al obtener estadísticas: ${response.statusCode}',
        );
      }
    } catch (e) {
      throw Exception('Error de conexión: $e');
    }
  }

  Future<TimelineData> getTimeline({
    required TimePeriod period,
    DateTime? startDate,
    DateTime? endDate,
  }) async {
    try {
      final token = await _getToken();

      final queryParams = {
        'period': period.value,
        if (startDate != null)
          'start_date': startDate.toIso8601String().split('T')[0],
        if (endDate != null)
          'end_date': endDate.toIso8601String().split('T')[0],
      };

      final uri = Uri.parse(
        '$apiUrl/admin/stats/timeline',
      ).replace(queryParameters: queryParams);

      final response = await http.get(
        uri,
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        final jsonResponse = json.decode(response.body);
        return TimelineData.fromJson(
          jsonResponse,
          period.value,
          startDate ??
              DateTime.now().subtract(Duration(days: period.defaultDays)),
          endDate ?? DateTime.now(),
        );
      } else {
        throw Exception('Error al obtener timeline: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Error de conexión: $e');
    }
  }

  Future<Map<String, dynamic>> getUsersList({
    int limit = 50,
    int offset = 0,
    String? search,
    bool? hasSubscription,
    String sortBy = 'created_at',
    String order = 'DESC',
  }) async {
    try {
      final token = await _getToken();

      final queryParams = {
        'limit': limit.toString(),
        'offset': offset.toString(),
        if (search != null && search.isNotEmpty) 'search': search,
        if (hasSubscription != null)
          'has_subscription': hasSubscription.toString(),
        'sort_by': sortBy,
        'order': order,
      };

      final uri = Uri.parse(
        '$apiUrl/admin/stats/users/list',
      ).replace(queryParameters: queryParams);

      final response = await http.get(
        uri,
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        final jsonResponse = json.decode(response.body);
        return {
          'users':
              (jsonResponse['data'] as List<dynamic>)
                  .map((e) => UserListItem.fromJson(e as Map<String, dynamic>))
                  .toList(),
          'total': jsonResponse['total'] ?? 0,
        };
      } else {
        throw Exception('Error al obtener usuarios: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Error de conexión: $e');
    }
  }

  Future<Map<String, dynamic>> getSubscriptionsHistory({
    int limit = 50,
    int offset = 0,
    int? userId,
    int? planId,
    bool activeOnly = false,
  }) async {
    try {
      final token = await _getToken();

      final queryParams = {
        'limit': limit.toString(),
        'offset': offset.toString(),
        if (userId != null) 'user_id': userId.toString(),
        if (planId != null) 'plan_id': planId.toString(),
        if (activeOnly) 'active_only': 'true',
      };

      final uri = Uri.parse(
        '$apiUrl/admin/stats/subscriptions/history',
      ).replace(queryParameters: queryParams);

      final response = await http.get(
        uri,
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        final jsonResponse = json.decode(response.body);
        return {
          'subscriptions':
              (jsonResponse['data'] as List<dynamic>)
                  .map(
                    (e) => SubscriptionHistoryItem.fromJson(
                      e as Map<String, dynamic>,
                    ),
                  )
                  .toList(),
          'total': jsonResponse['total'] ?? 0,
        };
      } else {
        throw Exception('Error al obtener historial: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Error de conexión: $e');
    }
  }
}
