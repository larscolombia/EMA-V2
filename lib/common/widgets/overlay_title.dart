import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';


class OverlayTitle extends StatelessWidget {
  final String title;
  final String subtitle;

  const OverlayTitle({super.key, 
    required this.title,
    required this.subtitle,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.start,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: AppStyles.title2White,
        ),

        const SizedBox(height: 8),

        Text(
          subtitle,
          style: AppStyles.subtitle,
        ),
      ],
    );
  }
}
