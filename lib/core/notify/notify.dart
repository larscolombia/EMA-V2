// ignore_for_file: avoid_print

import 'package:flutter/material.dart';
import 'package:get/get_core/src/get_main.dart';
import 'package:get/get_navigation/get_navigation.dart';

enum NotifyType { error, info, message, success, warning, }


class Notify {
  static void snackbar(String title, String message, [NotifyType type = NotifyType.message]) {

    Color color;

    switch (type) {
      case NotifyType.error:
        color = Colors.red;
        break;
      case NotifyType.info:
        color = Colors.blue;
        break;
      case NotifyType.message:
        color = Colors.black;
        break;
      case NotifyType.success:
        color = Colors.green;
        break;
      case NotifyType.warning:
        color = Colors.orange;
        break;
    }

    Get.snackbar(title, message, backgroundColor: color);
  }
}
