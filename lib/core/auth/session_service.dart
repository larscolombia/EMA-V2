import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:get/get.dart';

import '../core.dart';

class SessionService extends GetxService {
  final FlutterSecureStorage _storage = const FlutterSecureStorage();
  final LaravelAuthService _authService = Get.find<LaravelAuthService>();
  final UserService _userService = Get.find<UserService>();

  // Tiempo máximo de sesión en días (ajustar según necesidades)
  static const int _maxSessionDays = 30;

  Future<void> initSession() async {
    try {
      var token = await _storage.read(key: 'auth_token');
      final lastSessionStr = await _storage.read(key: 'last_session');

      // Si no hay token en almacenamiento, intentamos obtenerlo del usuario actual
      if ((token == null || token.isEmpty) &&
          _userService.currentUser.value.authToken.isNotEmpty) {
        token = _userService.currentUser.value.authToken;
      }

      if (token != null && token.isNotEmpty && lastSessionStr != null) {
        try {
          final lastSession = DateTime.parse(lastSessionStr);
          final now = DateTime.now();

          // Verificar si la sesión ha expirado
          if (now.difference(lastSession).inDays > _maxSessionDays) {
            await clearSession();
            return;
          }

          // Actualizar timestamp de última sesión
          await _storage.write(
            key: 'last_session',
            value: now.toIso8601String(),
          );

          // Obtener usuario actual y actualizar estado
          final user = await _authService.getUser(token);
          await _userService.setCurrentUser(user);

          // Persistir el token actualizado
          await _storage.write(key: 'auth_token', value: user.authToken);
        } catch (e) {
          try {
            final localUser = _userService.currentUser.value;
            await _userService.setCurrentUser(localUser);
          } catch (e2) {
            await clearSession();
          }
        }
      }
    } catch (e) {
      await clearSession();
    }
  }

  Future<void> clearSession() async {
    // Guardamos el email recordado antes de limpiar
    final rememberedEmail = await _storage.read(key: 'remembered_email');

    // Limpiamos todos los datos de sesión
    await _storage.deleteAll();

    // Restauramos el email recordado si existía
    if (rememberedEmail != null) {
      await _storage.write(key: 'remembered_email', value: rememberedEmail);
    }

    // Limpiamos el usuario actual
    await _userService.clearCurrentUser();
  }
}
