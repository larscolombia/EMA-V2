import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart'
    as quizzes;
import 'package:ema_educacion_medica_avanzada/app/quizzes/widgets/question_input.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class QuestionInputTrueFalse extends QuestionInput {
  final Rx<bool?> valueOption = Rx(null);
  final bool isDisabled;

  QuestionInputTrueFalse({
    super.key,
    required super.onAnswer,
    required super.question,
    this.isDisabled = false,
  });

  final ButtonStyle _selected = ButtonStyle(
    backgroundColor: WidgetStateProperty.all(Colors.blue),
    foregroundColor: WidgetStateProperty.all(Colors.white),
  );

  @override
  Widget build(BuildContext context) {
    Widget trueFalseButtons = Obx(() {
      final isTrue = valueOption.value != null && valueOption.value!;
      final isFalse = valueOption.value != null && !valueOption.value!;

      return Row(
        mainAxisAlignment: MainAxisAlignment.start,
        children: [
          TextButton(
            onPressed: isDisabled
                ? null
                : () {
                    valueOption.value = true;
                  },
            style: isTrue ? _selected : null,
            child: Text('Verdadero'),
          ),

          SizedBox(width: 16),

          TextButton(
            onPressed: isDisabled
                ? null
                : () {
                    valueOption.value = false;
                  },
            style: isFalse ? _selected : null,
            child: Text('Falso'),
          ),
        ],
      );
    });

    return quizzes.QuestionInputContainer(
      onPressed: isDisabled
          ? null
          : () {
              if (valueOption.value == null) return;

              final answer = AnswerModel.trueFalse(valueOption.value!);
              onAnswer(answer);
            },
      child: trueFalseButtons,
    );
  }
}
