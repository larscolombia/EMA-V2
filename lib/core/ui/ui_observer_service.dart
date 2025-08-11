import 'dart:async';

import 'package:flutter_keyboard_visibility/flutter_keyboard_visibility.dart';
import 'package:get/get.dart';


class UiObserverService extends GetxService {
  Rx<bool> isKeyboardVisible = false.obs;
  late KeyboardVisibilityController keyboardVisibilityController;
  late StreamSubscription<bool> keyboardSubscription;

  UiObserverService();

  Future<void> init() async {
    keyboardVisibilityController = KeyboardVisibilityController();

    isKeyboardVisible.value = keyboardVisibilityController.isVisible;

    keyboardSubscription = keyboardVisibilityController.onChange.listen((bool visible) {
      isKeyboardVisible.value = visible;
    });

    // super.onInit();
  }

  @override
  void onClose() {
    keyboardSubscription.cancel(); // Cancela la suscripción al cerrar el servicio
    super.onClose();
  }
}
