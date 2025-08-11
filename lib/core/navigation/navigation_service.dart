import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:get/get.dart';


class NavigationService extends GetxService {
  final Rx<EmaOverlayRoute> currentRoute = EmaOverlayRoute.empty().obs;

  NavigationService();

  void back() {
    currentRoute.value = EmaOverlayRoute.empty();
    Get.back();
  }

  void goTo(OverlayRoutes overlayRoute) {
    final route = AppOverlays.overlayRoutes.firstWhere(
      (route) => route.name == overlayRoute.name,
      orElse: () => EmaOverlayRoute.empty(),
    );

    currentRoute.value = route;

    Get.to(
      () => OverlayLayout(),
      opaque: false,
      fullscreenDialog: true,
      transition: Transition.fadeIn,
      duration: const Duration(milliseconds: 150),
    );
  }

  void show(OverlayRoutes newRoute) {
    currentRoute.value = AppOverlays.overlayRoutes.firstWhere(
      (route) => route.name == newRoute.name,
      orElse: () => EmaOverlayRoute.empty(),
    );
  }
}
