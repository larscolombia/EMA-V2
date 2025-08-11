import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_response_model.dart';
import 'package:flutter/material.dart';

abstract class QuestionInput extends StatelessWidget {
  final Function(AnswerModel) onAnswer;
  final QuestionResponseModel question;

  const QuestionInput({
    super.key,
    required this.onAnswer,
    required this.question,
  });
}

class QuestionInputContainer extends StatelessWidget {
  final VoidCallback? onPressed;
  final Widget child;
  final double opacity;

  const QuestionInputContainer({
    super.key,
    required this.child,
    this.onPressed,
    this.opacity = 1.0,
  });

  @override
  Widget build(BuildContext context) {
    return Opacity(
      opacity: opacity,
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
        margin: const EdgeInsets.only(bottom: 8),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: Colors.grey.shade300, width: 1),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            child,
            if (onPressed != null)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Align(
                  alignment: Alignment.centerRight,
                  child: TextButton(
                    onPressed: onPressed,
                    style: TextButton.styleFrom(
                      minimumSize: Size(100, 36),
                      backgroundColor:
                        onPressed == null
                          ? Colors.grey.shade300
                          : Colors.blue,
                      foregroundColor: Colors.white,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                    child: Text('Enviar'),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
