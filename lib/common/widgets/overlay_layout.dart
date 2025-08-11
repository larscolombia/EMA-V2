import 'dart:ui';

import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class OverlayLayout extends StatefulWidget {

  const OverlayLayout({
    super.key,
  });

  @override
  State<OverlayLayout> createState() => _OverlayLayoutState();
}

class _OverlayLayoutState extends State<OverlayLayout> with SingleTickerProviderStateMixin {
  final navigationService = Get.find<NavigationService>();

  late AnimationController _animationController;
  late Animation<double> _fadeAnimation;
  late Animation<Offset> _slideAnimation;

  @override
  void initState() {
    super.initState();

    _animationController = AnimationController(
      duration: const Duration(milliseconds: 300), // Duración de la animación (ajusta a tu gusto)
      vsync: this, //  'this' se refiere al TickerProviderStateMixin
    );

    // Animación de Fade (opacidad de 0 a 1)
    _fadeAnimation = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(
        parent: _animationController,
        curve: Curves.easeInOut, // Puedes elegir otra curva
      ),
    );

    // Animación de Slide (desde abajo (Offset(0, 1)) hasta su posición original (Offset.zero))
    _slideAnimation = Tween<Offset>(begin: const Offset(0, 0.3), end: Offset.zero).animate(
      CurvedAnimation(
        parent: _animationController,
        curve: Curves.easeInOut, // Misma curva para consistencia
      ),
    );

    // Iniciar la animación con un retraso de 100ms
    Future.delayed(const Duration(milliseconds: 100), () {
      if (mounted) { // Verifica si el widget todavía está montado antes de animar
        _animationController.forward(); // Inicia la animación
      }
    });
  }

  @override
  void dispose() {
    _animationController.dispose(); // ¡Siempre libera el AnimationController!
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        // Overlay
        Positioned.fill(
          child: Material(
            color: Colors.transparent,
            child: BackdropFilter(
              filter: ImageFilter.blur(sigmaX: 7, sigmaY: 7),
              child: Container(
                decoration: BoxDecoration(
                  color: AppStyles.primary900.withAlpha(90), // 255 * 0.35
                  backgroundBlendMode: BlendMode.srcOver,
                  borderRadius: BorderRadius.circular(10),
                ),
              ),
            ),
          ),
        ),

        // Scaffold
        Positioned.fill(
          child: FadeTransition(
            opacity: _fadeAnimation,
            child: SlideTransition(
              position: _slideAnimation,
              child: Scaffold(
                backgroundColor: Colors.transparent,
                body: Obx(() {
                  final route = navigationService.currentRoute.value;
                  final topArea = route.topArea?.call();
                  final topList = route.topList?.call();
                  final bottomArea = route.bottomArea();
                  
                  return GestureDetector(
                    onTap: navigationService.back,
                    behavior: HitTestBehavior.opaque,
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.end,
                      mainAxisSize: MainAxisSize.max,
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                    
                        // topArea
                        if (topArea != null)
                        Container(
                          padding: const EdgeInsets.all(16),
                          child: topArea,
                        ),
                    
                        // topList
                        if (topList != null)
                        Expanded(
                          child: Container(
                            padding: const EdgeInsets.all(16),
                            child: topList,
                          ),
                        ),
                    
                        // bottomArea
                        Container(
                          padding: const EdgeInsets.all(16),
                          decoration: BoxDecoration(color: Colors.white),
                          child: bottomArea,
                        ),
                      ],
                    ),
                  );
                }),
              ),
            ),
          ),
        ),

        // Close button - top left
        Positioned(
          top: 26,
          left: 8,
          child: IconButton(
            icon: AppIcons.closeSquare(color: AppStyles.whiteColor),
            onPressed: navigationService.back,
          ),
        ),
      ],
    );
  }
}
