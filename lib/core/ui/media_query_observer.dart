import 'package:flutter/material.dart';


class MediaQueryObserver extends WidgetsBindingObserver {
  void Function(AppLifecycleState)? onAppLifeCycleChange;
  VoidCallback? onMetricsChange;

  MediaQueryObserver({
    this.onAppLifeCycleChange,
    this.onMetricsChange
  });

  @override
  void didChangeMetrics() {
    onMetricsChange?.call();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    onAppLifeCycleChange?.call(state);
  }
}

// https://gemini.google.com/app/4e26abd5e0b79d38?hl=es
// class ConnectivityObserver extends GetxController {
//   var isConnected = true.obs; // Variable reactiva para el estado de conexión
//   @override
//   void onInit() {
//     super.onInit();
//     _checkConnectivity();
//     Connectivity().onConnectivityChanged.listen((ConnectivityResult result) {
//       _updateConnectionStatus(result);
//     });
//   }
//   void _checkConnectivity() async {
//     var connectivityResult = await (Connectivity().checkConnectivity());
//     _updateConnectionStatus(connectivityResult);
//   }
//   void _updateConnectionStatus(ConnectivityResult result) {
//     if (result == ConnectivityResult.none) {
//       isConnected.value = false;
//     } else {
//       isConnected.value = true;
//     }
//   }
// }
