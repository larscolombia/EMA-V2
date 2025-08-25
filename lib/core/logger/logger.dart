// ignore_for_file: avoid_print

import 'package:flutter/foundation.dart';

/// Logger minimalista con gating en release.
/// En release sólo deja pasar warnings y errores.
class Logger {
  static bool get _allowVerbose => kDebugMode || kProfileMode;

  static void debug(String msg) { if (_allowVerbose) print('[D] $msg'); }
  static void info(String msg)  { if (_allowVerbose) print('[I] $msg'); }
  static void warn(String msg)  { print('[W] $msg'); }
  static void error(String msg) { print('[E] $msg'); }

  // Backwards compatibility wrappers
  static void log(String message) => info(message);
  static void mini(String message) => debug(message);
  static void objectValue(String object, String value) {
    if (!_allowVerbose) return;
    debug('$object => $value');
  }
}