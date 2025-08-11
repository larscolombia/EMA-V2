import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:get/get.dart';

class UserLocalDataService extends GetxService {
  final _storage = const FlutterSecureStorage();
  final _userKey = 'KEY_USER_DATA';

  Future<void> clear() async {
    await _storage.delete(key: _userKey);
  }

  Future<UserModel> load() async {
    final userData = await _storage.read(key: _userKey);

    if (userData != null) {
      return UserModel.fromJson(userData);
    } else {
      throw Exception('No user found');
    }
  }

  Future<void> save(UserModel user) async {
    try {
      // Primero limpiamos cualquier dato existente
      await clear();
      // Luego guardamos los nuevos datos
      await _storage.write(key: _userKey, value: json.encode(user.toMap()));
    } catch (e) {
      Notify.snackbar('Auth', 'Error al guardar los datos del usuario', NotifyType.error);
    }
  }

  Future<void> deleteAll() async {
    // Guardamos el email recordado antes de limpiar
    final rememberedEmail = await _storage.read(key: 'remembered_email');

    // Limpiamos todo
    await _storage.deleteAll();

    // Restauramos el email recordado si existía
    if (rememberedEmail != null) {
      await _storage.write(key: 'remembered_email', value: rememberedEmail);
    }
  }
}
