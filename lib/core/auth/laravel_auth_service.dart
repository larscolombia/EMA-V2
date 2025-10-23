import 'dart:convert';
import 'dart:io';

import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:get/get.dart';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';
import '../../app/profiles/repositories/statistics_repository.dart'; // Nueva importación
// auth_token_provider is re-exported via core/core.dart

class LaravelAuthService extends GetxService {
  // Cliente HTTP personalizado que ignora errores SSL en debug
  http.Client _createHttpClient() {
    if (kDebugMode) {
      final ioClient =
          HttpClient()
            ..badCertificateCallback = (
              X509Certificate cert,
              String host,
              int port,
            ) {
              print(
                '⚠️ WARNING: Ignoring SSL certificate error for $host (DEBUG MODE ONLY)',
              );
              return true; // Ignora errores de certificado en modo debug
            };
      return IOClient(ioClient);
    }
    return http.Client();
  }

  /// El método el usuario que inicia la sesión,
  /// captura el laravelToken y lo envía al ApiService
  Future<UserModel> login(String username, String password) async {
    try {
      final client = _createHttpClient();

      final body = jsonEncode({
        'email': username,
        'password': password,
        'remember': false,
      });
      final url = '$apiUrl/login';
      final headers = {HttpHeaders.contentTypeHeader: 'application/json'};

      print('🔐 LOGIN REQUEST:');
      print('URL: $url');
      print('Body: $body');

      final response = await client.post(
        Uri.parse(url),
        headers: headers,
        body: body,
      );

      print('🔐 LOGIN RESPONSE:');
      print('Status: ${response.statusCode}');
      print('Body: ${response.body}');

      switch (response.statusCode) {
        case 200:
          final responseBody = json.decode(response.body);
          final user = UserModel.fromLaravelApi(responseBody);
          // Persist auth token for later API calls/background restarts
          try {
            await AuthTokenProvider.instance.saveToken(user.authToken);
          } catch (_) {}
          return user;
        case 401:
          throw Exception('Contraseña Incorrecta');
        case 422:
          throw Exception('Usuario o Contraseña Incorrecta');
        default:
          throw Exception('Error inesperado: ${response.statusCode}');
      }
    } catch (e) {
      print('🔐 LOGIN ERROR: $e');
      throw Exception(e.toString());
    }
  }

  Future<void> logout(String token) async {
    try {
      final client = _createHttpClient();
      final url = '$apiUrl/logout';
      final headers = {
        HttpHeaders.contentTypeHeader: 'application/json',
        HttpHeaders.authorizationHeader: 'Bearer $token',
      };

      final response = await client.post(Uri.parse(url), headers: headers);

      if (response.statusCode != 200) {
        throw Exception(
          'Error inesperado durante el logout: ${response.statusCode}',
        );
      }

      // Limpiar la caché de estadísticas al cerrar sesión
      final statisticsRepo = StatisticsRepository();
      await statisticsRepo.clearCache();
    } catch (e) {
      throw Exception('Error durante el logout: $e');
    }
  }

  Future<void> register(Map<String, dynamic> formData) async {
    final client = _createHttpClient();
    final url = Uri.parse('$apiUrl/register');

    print('📝 REGISTER REQUEST:');
    print('URL: $url');
    print('Body: ${jsonEncode(formData)}');

    final response = await client.post(
      url,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: jsonEncode(formData),
    );

    print('📝 REGISTER RESPONSE:');
    print('Status: ${response.statusCode}');
    print('Body: ${response.body}');

    if (response.statusCode != 201) {
      throw Exception('Registration failed: ${response.body}');
    }
  }

  Future<void> forgotPassword(String email) async {
    final client = _createHttpClient();
    final url = Uri.parse('$apiUrl/password/forgot');

    final response = await client.post(
      url,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'email': email}),
    );

    if (response.statusCode != 200) {
      throw Exception('Forgot password request failed: ${response.body}');
    }
  }

  Future<UserModel> getUser(String token) async {
    final client = _createHttpClient();
    final url = Uri.parse('$apiUrl/session');
    final headers = {
      HttpHeaders.contentTypeHeader: 'application/json',
      HttpHeaders.authorizationHeader: 'Bearer $token',
    };

    final response = await client.get(url, headers: headers);

    if (response.statusCode == 200) {
      final responseBody = json.decode(response.body);
      return UserModel.fromLaravelApi(responseBody);
    } else {
      throw Exception('Error al obtener el usuario: ${response.statusCode}');
    }
  }
}
