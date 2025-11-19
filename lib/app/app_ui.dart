import 'package:ema_educacion_medica_avanzada/admin/config/admin_routes.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/cupertino.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:get/get.dart';

class AppUi extends StatelessWidget {
  const AppUi({super.key});

  @override
  Widget build(BuildContext context) {
    // En web, redirigir directamente al panel admin
    // En móvil, usar la pantalla de inicio normal
    final String initialRoute = kIsWeb ? AdminRoutes.login : Routes.start.name;

    return GetMaterialApp(
      debugShowCheckedModeBanner: false,
      getPages: AppPages.routes,
      initialRoute: initialRoute,
      theme: AppTheme.getTheme(),
      title: 'ema Educacion Medica Avanzada',
    );
  }
}
