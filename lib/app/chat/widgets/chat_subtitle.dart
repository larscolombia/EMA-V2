import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';

class ChatSubtitle extends StatelessWidget {
  final String name;

  const ChatSubtitle({
    super.key,
    required this.name,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppStyles.secondaryColor,
      ),
      child: Text(
        name,
        style: AppStyles.chatSubtitle,
      ),
    );
  }
}
