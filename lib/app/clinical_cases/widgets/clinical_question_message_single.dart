import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';

class ClinicalQuestionMessageSingle extends StatelessWidget {
  final QuestionResponseModel question;
  ClinicalQuestionMessageSingle({super.key, required this.question});

  final List<String> letters = const ['A.', 'B.', 'C.', 'D.', 'E.'];

  @override
  Widget build(BuildContext context) {
    final base = Theme.of(context).textTheme.bodyMedium;
    final questionStyle = base?.copyWith(
      fontSize: 16,
      height: 1.4,
      color: Colors.white,
      fontWeight: FontWeight.w600,
    );
    final optionStyle = base?.copyWith(
      fontSize: 16,
      height: 1.4,
      color: Colors.white,
    );

    int index = -1;
    final textOptions =
        question.options.map((option) {
          index++;
          return Padding(
            padding: const EdgeInsets.only(top: 8.0),
            child: Text(
              '${letters[index]} $option',
              style: optionStyle,
              textAlign: TextAlign.justify,
            ),
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
                Text(
                  question.question,
                  style: questionStyle,
                  textAlign: TextAlign.justify,
                ),
                ...textOptions,
              ],
            ),
          ),
        if (question.isAnswered)
          Align(
            alignment: Alignment.centerRight,
            child: Container(
              constraints: BoxConstraints(
                maxWidth: MediaQuery.of(context).size.width * 0.75,
              ),
              margin: const EdgeInsets.only(top: 0, bottom: 8, left: 48),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: AppStyles.whiteColor,
                border: Border.all(
                  color:
                      question.isCorrect == true
                          ? Colors.green
                          : (question.isCorrect == false
                              ? Colors.red
                              : AppStyles.primary900),
                  width: 2,
                ),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (question.isCorrect == true)
                    const Padding(
                      padding: EdgeInsets.only(right: 8.0),
                      child: Icon(
                        Icons.check_circle,
                        color: Colors.green,
                        size: 20,
                      ),
                    )
                  else if (question.isCorrect == false)
                    const Padding(
                      padding: EdgeInsets.only(right: 8.0),
                      child: Icon(Icons.cancel, color: Colors.red, size: 20),
                    ),
                  Expanded(
                    child: Text(
                      question.answerdString,
                      style: AppStyles.chatMessageUser,
                      textAlign: TextAlign.left,
                    ),
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}
