import 'dart:convert';
import 'dart:io';
import 'package:ema_educacion_medica_avanzada/admin/core/models/admin_user.dart';
import 'package:ema_educacion_medica_avanzada/config/constants/constants.dart';
import 'package:flutter/foundation.dart';
import 'package:get/get.dart';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class AdminAuthService extends GetxService {
  static const _storage = FlutterSecureStorage();
  static const _tokenKey = 'admin_auth_token';
  static const _userKey = 'admin_user_data';

  // Cache en memoria para evitar múltiples validaciones
  AdminUser? _cachedUser;
  DateTime? _lastValidation;
  static const _cacheValidityDuration = Duration(minutes: 5);

  http.Client _createHttpClient() {
    // En web, solo podemos usar http.Client() básico
    if (kIsWeb) {
      return http.Client();
    }

    // En plataformas nativas (móvil/desktop), podemos ignorar SSL en debug
    if (kDebugMode) {
      final ioClient =
          HttpClient()
            ..badCertificateCallback = (cert, host, port) {
              print('⚠️ [ADMIN] Ignoring SSL for $host (DEBUG MODE)');
              return true;
            };
      return IOClient(ioClient);
    }
    return http.Client();
  }

  Future<AdminUser> login(String email, String password) async {
    try {
      final client = _createHttpClient();
      final body = jsonEncode({
        'email': email,
        'password': password,
        'remember': true,
      });
      final url = '$apiUrl/login';
      final headers = {'Content-Type': 'application/json'};

      print('🔐 [ADMIN LOGIN] URL: $url');
      print('🔐 [ADMIN LOGIN] Email: $email');

      final response = await client.post(
        Uri.parse(url),
        headers: headers,
        body: body,
      );

      print('🔐 [ADMIN LOGIN] Status: ${response.statusCode}');
      print('🔐 [ADMIN LOGIN] Body: ${response.body}');

      if (response.statusCode == 200) {
        final responseBody = json.decode(response.body);
        final user = AdminUser.fromJson(responseBody);

        // Validar que sea super_admin
        if (!user.isSuperAdmin) {
          throw Exception(
            'Acceso denegado: Solo administradores pueden acceder al panel',
          );
        }

        // Guardar token y datos de usuario
        await _storage.write(key: _tokenKey, value: user.token);
        await _storage.write(key: _userKey, value: jsonEncode(user.toJson()));

        // Actualizar cache
        _cachedUser = user;
        _lastValidation = DateTime.now();

        return user;
      } else if (response.statusCode == 401) {
        throw Exception('Credenciales incorrectas');
      } else {
        throw Exception('Error inesperado: ${response.statusCode}');
      }
    } catch (e) {
      print('🔐 [ADMIN LOGIN ERROR]: $e');
      rethrow;
    }
  }

  Future<void> logout() async {
    try {
      final token = await _storage.read(key: _tokenKey);
      if (token != null) {
        final client = _createHttpClient();
        final url = '$apiUrl/logout';
        final headers = {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $token',
        };
        await client.post(Uri.parse(url), headers: headers);
      }
    } catch (e) {
      print('⚠️ [ADMIN LOGOUT ERROR]: $e');
    } finally {
      // Limpiar cache en memoria
      _cachedUser = null;
      _lastValidation = null;

      // Limpiar storage
      await _storage.delete(key: _tokenKey);
      await _storage.delete(key: _userKey);
    }
  }

  Future<AdminUser?> getCurrentUser({bool forceRefresh = false}) async {
    try {
      // Si tenemos cache válido y no se fuerza refresh, retornar cache
      if (!forceRefresh &&
          _cachedUser != null &&
          _lastValidation != null &&
          DateTime.now().difference(_lastValidation!) <
              _cacheValidityDuration) {
        print('🔐 [ADMIN] Usando usuario en caché');
        return _cachedUser;
      }

      final token = await _storage.read(key: _tokenKey);
      final userData = await _storage.read(key: _userKey);

      if (token == null || userData == null) {
        _cachedUser = null;
        _lastValidation = null;
        return null;
      }

      // Si no forzamos refresh, intentar usar datos guardados localmente
      if (!forceRefresh) {
        try {
          final userMap = json.decode(userData) as Map<String, dynamic>;
          final localUser = AdminUser.fromJson({
            'user': userMap,
            'token': token,
          });

          if (localUser.isSuperAdmin) {
            _cachedUser = localUser;
            _lastValidation = DateTime.now();
            print('🔐 [ADMIN] Usuario cargado desde storage local');
            return localUser;
          }
        } catch (e) {
          print('⚠️ [ADMIN] Error parseando usuario local: $e');
        }
      }

      // Validar token con el backend solo si es necesario
      print('🔐 [ADMIN] Validando sesión con backend...');
      final client = _createHttpClient();
      final url = Uri.parse('$apiUrl/session');
      final headers = {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $token',
      };

      final response = await client.get(url, headers: headers);

      if (response.statusCode == 200) {
        final responseBody = json.decode(response.body);
        final user = AdminUser.fromJson(responseBody);

        if (!user.isSuperAdmin) {
          await logout();
          return null;
        }

        // Actualizar cache
        _cachedUser = user;
        _lastValidation = DateTime.now();

        return user;
      } else {
        await logout();
        return null;
      }
    } catch (e) {
      print('⚠️ [ADMIN GET USER ERROR]: $e');
      // No hacer logout automático en caso de error de red
      // Solo retornar null
      return _cachedUser; // Devolver cache si hay error de red
    }
  }

  Future<String?> getToken() async {
    return await _storage.read(key: _tokenKey);
  }

  Future<bool> isAuthenticated() async {
    final user = await getCurrentUser();
    return user != null && user.isSuperAdmin;
  }
}
