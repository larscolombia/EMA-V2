import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/widgets/question_input.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/app_icons.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';


class QuestionInputText extends QuestionInput {
  final String title;
  final bool isDisabled;

  QuestionInputText({
    super.key,
    required super.onAnswer,
    required super.question,
    this.title = 'Mi respuesta es...',
    this.isDisabled = false,
  });

  final _outlineEnableBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(8),
    borderSide: BorderSide(
      color: Colors.transparent,
    ),
  );

  final _outlineFocusBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(8),
    borderSide: BorderSide(
      color: AppStyles.primary900,
    ),
  );

  @override
  Widget build(BuildContext context) {
    final textController = TextEditingController();
    final focusNode = FocusNode();

    final button = IconButton(
      onPressed: isDisabled
          ? null
          : () {
              final answer = AnswerModel.openEnded(textController.value.text);
              onAnswer(answer);

              textController.clear();
              focusNode.requestFocus();
            },
      icon: AppIcons.cornerDownLeft(
        height: 18,
        width: 18,
        color: AppStyles.primary900,
      ),
    );

    final inputDecoration = InputDecoration(
      label: Text(title),
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
      enabled: !isDisabled,
      keyboardType: TextInputType.text,
      maxLines: null,

      onFieldSubmitted: isDisabled
          ? null
          : (value) {
              final answer = AnswerModel.openEnded(textController.value.text);
              onAnswer(answer);

              textController.clear();
              focusNode.requestFocus();
            },

      onTapOutside: (event) {
        focusNode.unfocus();
      },
    );
  }
}
