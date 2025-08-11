import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';


class OutlineAiButton extends StatelessWidget {
  final String text;
  final VoidCallback onPressed;
  final bool enabled;

  OutlineAiButton({
    super.key,
    required this.text,
    required this.onPressed,
    this.enabled = true,
  });

  final textStyleEnabled = TextStyle(
    fontSize: 16,
    fontWeight: FontWeight.w600,
    color: AppStyles.primary900,
  );

  final textStyleDisabled = TextStyle(
    fontSize: 16,
    fontWeight: FontWeight.w600,
    color: AppStyles.grey150,
  );

  @override
  Widget build(BuildContext context) {
    // final stateController = WidgetStatesController();

    return Align(
      alignment: Alignment.center,
      child: OutlinedButton.icon(
        onPressed: enabled ? onPressed : null,
        // statesController:stateController,

        icon: AppIcons.startsAi(
          height: 24,
          width: 24,
          color: enabled
            ? AppStyles.primary900
            : AppStyles.grey150,
        ),

        label: Text(
          text,
          textAlign: TextAlign.center,
          style: enabled
            ? textStyleEnabled
            : textStyleDisabled,
        ),

        style: ButtonStyle(
          padding: WidgetStateProperty.all(
            EdgeInsets.symmetric(horizontal: 24),
          ),
          side: WidgetStateProperty.all(
            BorderSide(color: AppStyles.primary900),
          ),
        ),
      ),
    );
  }
}
