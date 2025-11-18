import 'package:ema_educacion_medica_avanzada/admin/core/services/admin_auth_service.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/auth/controllers/admin_auth_controller.dart';
import 'package:get/get.dart';

class AdminBindings extends Bindings {
  @override
  void dependencies() {
    // Servicio permanente en memoria (no se destruye al navegar)
    Get.put<AdminAuthService>(AdminAuthService(), permanent: true);

    // Controller permanente para evitar reinicializaciones
    Get.put<AdminAuthController>(AdminAuthController(), permanent: true);
  }
}
