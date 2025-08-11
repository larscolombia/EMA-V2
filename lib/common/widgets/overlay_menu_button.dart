import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';

class OverlayMenuButton extends StatelessWidget {
  final String title;
  final VoidCallback onPressed;
  final Widget icon;

  const OverlayMenuButton({
    super.key,
    required this.title,
    required this.onPressed,
    required this.icon,
  });

  @override
  Widget build(BuildContext context) {
    return TextButton(
      style: ButtonStyle(
        padding: WidgetStateProperty.all(const EdgeInsets.symmetric(horizontal: 0, vertical: 4)),
        shape: WidgetStateProperty.all(RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(14),
        )),
      ),
      onPressed: onPressed,
      child: Row(
        children: [
          icon,
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              title,
              style: AppStyles.menuBtn,
            ),
          ),
          AppIcons.chevronRight(height: 24, width: 24),
        ],
      ),
    );
  }
}
