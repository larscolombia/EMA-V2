import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';

import '../../core/auth/session_service.dart';

class StartController extends GetxController {
  final Rx<double> loading = 0.0.obs;

  @override
  void onInit() {
    _startApp();
    super.onInit();
  }

  void _startApp() async {
    await registerDependencies();

    final uiObserverService = Get.find<UiObserverService>();
    final dataBaseService = Get.find<DatabaseService>();
    final userService = Get.find<UserService>();
    final apiService = Get.find<ApiService>();
    final sessionService = Get.find<SessionService>();

    loading.value = 0.25;
    await uiObserverService.init();

    loading.value = 0.35;
    await dataBaseService.init();

    loading.value = 0.65;
    await userService.init();

    loading.value = 0.85;
    await apiService.init();

    await sessionService.initSession();
    loading.value = 1.0;

    if (userService.currentUser.value.id == 0) {
      Get.offAllNamed(Routes.login.name);
    } else {
      Get.offAllNamed(Routes.home.name);
    }
  }
}
