import 'dart:convert';
import 'dart:io';

import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';
import 'package:http/http.dart' as http;
import '../../app/profiles/repositories/statistics_repository.dart'; // Nueva importación

class LaravelAuthService extends GetxService {
  /// El método el usuario que inicia la sesión,
  /// captura el laravelToken y lo envía al ApiService
  Future<UserModel> login(String username, String password) async {
    try {
      final client = http.Client();

      final body = jsonEncode({'email': username, 'password': password});
      final url = '$apiUrl/login';
      final headers = {HttpHeaders.contentTypeHeader: 'application/json'};

      final response =
          await client.post(Uri.parse(url), headers: headers, body: body);

      switch (response.statusCode) {
        case 200:
          var responseBody = json.decode(response.body);

          return UserModel.fromLaravelApi(responseBody);

        case 401:
          throw Exception('Contraseña Incorrecta');

        case 422:
          throw Exception('Usuario o Contraseña Incorrecta');

        default:
          throw Exception('Error inesperado: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception(e.toString());
    }
  }

  Future<void> logout(String token) async {
    try {
      final client = http.Client();
      final url = '$apiUrl/logout';
      final headers = {
        HttpHeaders.contentTypeHeader: 'application/json',
        HttpHeaders.authorizationHeader: 'Bearer $token',
      };

      final response = await client.post(Uri.parse(url), headers: headers);

      if (response.statusCode != 200) {
        throw Exception(
            'Error inesperado durante el logout: ${response.statusCode}');
      }

      // Limpiar la caché de estadísticas al cerrar sesión
      final statisticsRepo = StatisticsRepository();
      await statisticsRepo.clearCache();
    } catch (e) {
      throw Exception('Error durante el logout: $e');
    }
  }

  Future<void> register(Map<String, dynamic> formData) async {
    final url = Uri.parse('$apiUrl/register');

    final response = await http.post(
      url,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: jsonEncode(formData),
    );

    if (response.statusCode != 201) {
      throw Exception('Registration failed: ${response.body}');
    }
  }

  Future<void> forgotPassword(String email) async {
    final url = Uri.parse('$apiUrl/password/forgot');

    final response = await http.post(
      url,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'email': email}),
    );

    if (response.statusCode != 200) {
      throw Exception('Forgot password request failed: ${response.body}');
    }
  }

  Future<UserModel> getUser(String token) async {
    final url = Uri.parse('$apiUrl/user');
    final headers = {
      HttpHeaders.contentTypeHeader: 'application/json',
      HttpHeaders.authorizationHeader: 'Bearer $token',
    };

    final response = await http.get(url, headers: headers);

    if (response.statusCode == 200) {
      final responseBody = json.decode(response.body);
      if (responseBody['success'] == true && responseBody['data'] != null) {
        // Adjuntar el token al usuario para que se mantenga la sesión
        responseBody['data']['token'] = token;
        return UserModel.fromLaravelApi({
          'user': responseBody['data'],
          'token': token,
        });
      } else {
        throw Exception('Respuesta no válida del servidor');
      }
    } else {
      throw Exception('Error al obtener el usuario: ${response.statusCode}');
    }
  }
}
