import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart'
    as quizzes;
import 'package:ema_educacion_medica_avanzada/app/quizzes/widgets/question_input.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class QuestionInputSingleChoice extends QuestionInput {
  final Rx<String> selectedOption = Rx('');
  final bool isDisabled;

  QuestionInputSingleChoice({
    super.key,
    required super.onAnswer,
    required super.question,
    this.isDisabled = false,
  });

  final ButtonStyle _normal = ButtonStyle(
    padding: WidgetStateProperty.all(EdgeInsets.all(4)),
  );

  final ButtonStyle _selected = ButtonStyle(
    backgroundColor: WidgetStateProperty.all(Colors.blue),
    foregroundColor: WidgetStateProperty.all(Colors.white),
    padding: WidgetStateProperty.all(EdgeInsets.all(4)),
  );

  final ButtonStyle _disabled = ButtonStyle(
    backgroundColor: WidgetStateProperty.all(Colors.grey.shade200),
    foregroundColor: WidgetStateProperty.all(Colors.grey.shade500),
    padding: WidgetStateProperty.all(EdgeInsets.all(4)),
  );

  @override
  Widget build(BuildContext context) {
    final numOptions = question.options.length;
    final letterOptions = AnswerModel.letters.take(numOptions).toList();

    return Obx(
      () => quizzes.QuestionInputContainer(
        onPressed: () {
          if (isDisabled || selectedOption.value.isEmpty) return;
          final answer = AnswerModel.singleChoice(
            selectedOption.value,
            question.options,
          );
          onAnswer(answer);
        },
        // Apply opacity to the entire container when disabled
        child: Row(
          mainAxisAlignment: MainAxisAlignment.start,
          children:
              letterOptions.map((l) {
                final isSelected = selectedOption.value == l;

                // Choose the button style based on state
                ButtonStyle buttonStyle;
                if (isDisabled) {
                  buttonStyle = _disabled;
                } else if (isSelected) {
                  buttonStyle = _selected;
                } else {
                  buttonStyle = _normal;
                }

                return IconButton(
                  onPressed:
                      isDisabled
                          ? null // Disable button when isDisabled is true
                          : () {
                            selectedOption.value = l;
                          },
                  isSelected: isSelected,
                  style: buttonStyle,
                  visualDensity: VisualDensity.compact,
                  icon: Text(
                    l,
                    style:
                        isSelected && !isDisabled
                            ? TextStyle(color: Colors.white)
                            : null,
                  ),
                );
              }).toList(),
        ),
      ),
    );
  }
}
