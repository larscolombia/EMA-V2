import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';

class QuestionMessageSingle extends StatelessWidget {
  final QuestionResponseModel question;

  QuestionMessageSingle({super.key, required this.question});

  final List<String> letters = ['A.', 'B.', 'C.', 'D.', 'E.'];

  @override
  Widget build(BuildContext context) {
    int index = -1;

    final textOptions =
        question.options.map((option) {
          index++;
          return Text(
            '${letters[index]} $option',
            style: AppStyles.chatMessageAi,
            textAlign: TextAlign.left,
          );
        }).toList();

    return Column(
      mainAxisSize: MainAxisSize.max,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (question.question.isNotEmpty)
          Container(
            margin: const EdgeInsets.only(top: 8, bottom: 8, right: 24),
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: AppStyles.primary900,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(question.question, style: AppStyles.chatMessageAi),
                ...textOptions,
              ],
            ),
          ),

        if (question.isAnswered)
          Container(
            margin: const EdgeInsets.only(top: 0, bottom: 8, left: 24),
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: AppStyles.whiteColor,
              border: Border.all(color: AppStyles.primary900, width: 1),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              question.answerdString,
              style: AppStyles.chatMessageUser,
              textAlign: TextAlign.right,
            ),
          ),
      ],
    );
  }
}
