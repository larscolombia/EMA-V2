import 'package:flutter/material.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';

class ValidationBar extends StatelessWidget {
  final String label;
  final bool isValid;

  const ValidationBar({
    super.key,
    required this.label,
    required this.isValid,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Container(
          height: 4.0,
          width: 50.0,
          decoration: BoxDecoration(
            color: isValid ? Colors.green : AppStyles.blackColor,
            borderRadius: BorderRadius.circular(2.0),
          ),
        ),
        const SizedBox(height: 4.0),
        Text(
          label,
          style: TextStyle(
            fontSize: 12.0,
            color: isValid ? Colors.green : AppStyles.blackColor,
          ),
        ),
      ],
    );
  }
}
