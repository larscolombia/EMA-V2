import 'package:flutter/material.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';

class PremiumBanner extends StatelessWidget {
  const PremiumBanner({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 5, horizontal: 12),
      margin: const EdgeInsets.symmetric(horizontal: 28),
      decoration: BoxDecoration(
        color: AppStyles.primary900,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(
            Icons.star,
            color: Colors.yellow[700],
            size: 22,
          ),
          const SizedBox(width: 8),
          const Expanded(
            child: Text(
              'Genial, eres Premium!',
              style: TextStyle(
                color: AppStyles.whiteColor,
                fontSize: 15,
                fontWeight: FontWeight.w800,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
