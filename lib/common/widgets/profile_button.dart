import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';

class ProfileButton extends StatelessWidget {
  final String label;
  final Widget icon;
  final Color btnColor;
  final int flex;

  const ProfileButton({
    super.key,
    required this.label,
    required this.icon,
    this.btnColor = AppStyles.tertiaryColor,
    this.flex = 0,
  });

  @override
  Widget build(BuildContext context) {
    return Expanded(
      flex: flex,
      child: FilledButton.tonal(
        onPressed: () {},
        style: ButtonStyle(
          backgroundColor: WidgetStateProperty.all(btnColor),
          shape: WidgetStateProperty.all(RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          )),
          padding: WidgetStateProperty.all(
            EdgeInsets.all(24),
          ),
        ),
        child: Column(
          children: [
            icon,
            Text(
              label,
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: Colors.white,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
