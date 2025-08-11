import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/app_icons.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get_state_manager/src/rx_flutter/rx_obx_widget.dart';

class ClinicalCaseChatFieldBox extends StatelessWidget {
  final ClinicalCaseController controller;
  final String title;

  ClinicalCaseChatFieldBox({super.key, required this.controller, this.title = 'Escribe...'});

  final _outlineEnableBorder = OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: BorderSide(color: Colors.transparent));

  final _outlineFocusBorder = OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: BorderSide(color: AppStyles.primary900));

  @override
  Widget build(BuildContext context) {
    final textController = TextEditingController();
    final focusNode = FocusNode();

    return Padding(
      padding: EdgeInsets.symmetric(horizontal: 20, vertical: 8),
      child: Obx(() {
        final typing = controller.isTyping.value;

        final button = IconButton(
          onPressed:
              typing
                  ? null
                  : () {
                    controller.sendMessage(textController.value.text);
                    textController.clear();
                    focusNode.requestFocus();
                  },
          icon: AppIcons.cornerDownLeft(height: 18, width: 18, color: AppStyles.primary900),
        );

        final inputDecoration = InputDecoration(
          label: Text(title),
          contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 12),
          enabledBorder: _outlineEnableBorder,
          focusedBorder: _outlineFocusBorder,
          floatingLabelBehavior: FloatingLabelBehavior.never,
          suffixIcon: button,
          filled: true,
        );

        return TextFormField(
          autocorrect: false,
          focusNode: focusNode,
          controller: textController,
          decoration: inputDecoration,
          enabled: !typing,
          keyboardType: TextInputType.text,
          maxLines: null,
          onFieldSubmitted:
              typing
                  ? null
                  : (value) {
                    controller.sendMessage(value);
                    textController.clear();
                    focusNode.requestFocus();
                  },
          onTapOutside: (event) {
            focusNode.unfocus();
          },
        );
      }),
    );
  }
}
