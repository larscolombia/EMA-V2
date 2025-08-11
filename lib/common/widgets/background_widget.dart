import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';

class BackgroundWidget extends StatelessWidget {
  final Widget child;
  final Color color;

  const BackgroundWidget({
    super.key,
    required this.child,
    this.color = AppStyles.primary900,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: color,
      body: Stack(
        fit: StackFit.expand, // Asegura que el Stack ocupe toda la pantalla.
        children: [
          // Imagen de fondo inferior
          Positioned.fill(
            child: Align(
              alignment: Alignment.bottomCenter,
              child: Image.asset(
                'assets/images/start_footer.png',
                fit: BoxFit.fill,
                width: double.infinity,
              ),
            ),
          ),
          // Texto inferior derecho
          const Positioned(
            bottom: 30.0,
            right: 16.0,
            child: Text(
              'Powered with AI by LARS',
              style: TextStyle(
                fontSize: 12.0,
                fontWeight: FontWeight.w500,
                color: Colors.white,
              ),
            ),
          ),
          // Contenido principal con un Padding inferior (para no tapar el footer)
          Padding(
            padding: const EdgeInsets.only(bottom: 20),
            child: child,
          ),
        ],
      ),
    );
  }
}
