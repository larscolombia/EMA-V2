import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/cupertino.dart';
import 'package:get/get.dart';

class AppUi extends StatelessWidget {
  const AppUi({super.key});

  @override
  Widget build(BuildContext context) {
    return GetMaterialApp(
      debugShowCheckedModeBanner: false,
      getPages: AppPages.routes,
      initialRoute: Routes.start.name,
      theme: AppTheme.getTheme(),
      title: 'ema Educacion Medica Avanzada',
    );
  }
}
