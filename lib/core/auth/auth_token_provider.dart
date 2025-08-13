import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:get/get.dart';

import '../core.dart';

/// Centralized helper to obtain and persist auth tokens.
class AuthTokenProvider {
  AuthTokenProvider._();
  static final AuthTokenProvider instance = AuthTokenProvider._();

  final _storage = const FlutterSecureStorage();

  /// Returns the best-known token, preferring in-memory user state,
  /// then secure storage. Never throws; returns empty string if none.
  Future<String> getToken() async {
    try {
      final userService =
          Get.isRegistered<UserService>() ? Get.find<UserService>() : null;
      final mem = userService?.currentUser.value.authToken ?? '';
      if (mem.isNotEmpty) return mem;
      final stored = await _storage.read(key: 'auth_token') ?? '';
      return stored;
    } catch (_) {
      return '';
    }
  }

  /// Persists the token to secure storage and updates in-memory user if present.
  Future<void> saveToken(String token) async {
    try {
      if (token.isEmpty) return;
      await _storage.write(key: 'auth_token', value: token);
      if (Get.isRegistered<UserService>()) {
        final userService = Get.find<UserService>();
        final current = userService.currentUser.value;
        if (current.id != 0 && current.authToken != token) {
          await userService.setCurrentUser(current.copyWith(authToken: token));
        }
      }
    } catch (_) {
      // ignore
    }
  }
}
