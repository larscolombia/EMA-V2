import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';
import '../auth/session_service.dart';

class UserService extends GetxController {
  // Cambiado de GetxService a GetxController
  final _laravelAuthService = Get.find<LaravelAuthService>();
  final _userLocalData = Get.find<UserLocalDataService>();
  Rx<UserModel> currentUser = UserModel.unknow().obs;

  Future<void> init() async {
    try {
      final storedUser = await _userLocalData.load();
      currentUser.value = storedUser;
    } catch (e) {
      currentUser.value = UserModel.unknow();
    }
  }

  Future<void> clearCurrentUser() async {
    currentUser.value = UserModel.unknow();
    await _userLocalData.clear();
  }

  Future<void> setCurrentUser(UserModel user) async {
    // Actualizar el estado
    currentUser.value = user;

    // Persistir los datos
    await _userLocalData.save(user);

    // Notificar cambios
    update();
  }

  UserModel getProfileData() => currentUser.value;

  void login(String username, String password) async {
    try {
      currentUser.value = await _laravelAuthService.login(username, password);

      await _userLocalData.save(currentUser.value);

      Notify.snackbar('Login', 'Accedio correctamente', NotifyType.success);

      Get.offAllNamed(Routes.home.name);
    } catch (e) {
      Notify.snackbar('Logout', e.toString(), NotifyType.error);
      currentUser.value = UserModel.unknow();
    }
  }

  void logout() async {
    // Navegar al login
    Get.offAllNamed(Routes.login.name);

    // Intentar cerrar sesión en backend, ignorar errores
    try {
      await _laravelAuthService.logout(currentUser.value.authToken);
    } catch (e) {
      // Error al cerrar sesión en backend, proceder de todas formas
    }

    // Limpiar datos locales y de sesión
    await _userLocalData.deleteAll();
    await Get.find<SessionService>().clearSession();
    currentUser.value = UserModel.unknow();
    await _userLocalData.clear();

    // Notificar cierre de sesión exitoso
    Notify.snackbar(
        'Cerrar sesión', 'Se cerró la sesión exitosamente', NotifyType.success);
  }

  void showProfileView() {
    final userId = currentUser.value.id;
    final path = currentUser.value.id > 0
        ? Routes.profile.path(userId.toString())
        : Routes.login.name;

    Get.toNamed(path);
  }
}
