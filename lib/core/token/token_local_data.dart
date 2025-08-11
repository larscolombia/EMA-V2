import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';


class TokenLocalData {
  final _storage = const FlutterSecureStorage();
  final _userKey = 'AUTH_TOKEN_DATA';

  Future<void> clear() async {
    _storage.delete(key: _userKey);
  }

  Future<LaravelToken> load() async {
    final tokenData = await _storage.read(key: _userKey);

    if (tokenData != null) {
      return LaravelToken.fromJson(tokenData);
    } else {
      throw Exception('No token found');
    }
  }

  Future<void> save(LaravelToken token) async {
    try {
      await _storage.write(key: _userKey, value: json.encode(token.toString()));
    } catch (e) {
      Notify.snackbar('Auth', 'Error al guardar los datos del token', NotifyType.error);
    }
  }
}
