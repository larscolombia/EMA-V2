import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';

class QuestionInputContainer extends StatelessWidget {
  final Widget child;
  final VoidCallback? onPressed;

  const QuestionInputContainer({
    super.key,
    required this.child,
    this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: AppStyles.grey220,
        borderRadius: BorderRadius.circular(8),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: Row(
        children: [
          Expanded(child: child),

          const SizedBox(width: 8),

          IconButton(
            onPressed: onPressed,
            icon: AppIcons.cornerDownLeft(
              height: 18,
              width: 18,
              color: AppStyles.primary900,
            ),
          ),
        ],
      ),
    );
  }
}
