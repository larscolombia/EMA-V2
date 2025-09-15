import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';

class QuestionFeedbackList extends StatelessWidget {
  final List<QuestionResponseModel> questions;

  const QuestionFeedbackList({super.key, required this.questions});

  @override
  Widget build(BuildContext context) {
    if (questions.isEmpty) {
      return const SizedBox.shrink();
    }

    final base = Theme.of(context).textTheme.bodyMedium;
    final body = base?.copyWith(fontSize: 16, height: 1.4);
    final purpleBold = base?.copyWith(
      fontSize: 16,
      height: 1.4,
      fontWeight: FontWeight.w700,
      color: AppStyles.primary900,
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const SizedBox(height: 8),
        for (var i = 0; i < questions.length; i++) ...[
          _QuestionBlock(
            index: i + 1,
            question: questions[i],
            purpleBold: purpleBold,
            body: body,
          ),
          if (i < questions.length - 1) const Divider(height: 24),
        ],
      ],
    );
  }
}

class _QuestionBlock extends StatelessWidget {
  final int index;
  final QuestionResponseModel question;
  final TextStyle? purpleBold;
  final TextStyle? body;

  const _QuestionBlock({
    required this.index,
    required this.question,
    required this.purpleBold,
    required this.body,
  });

  @override
  Widget build(BuildContext context) {
    final statusText = (question.isCorrect == true) ? 'Correcta' : 'Incorrecta';
    final userAns = (question.answerdString).trim();
    final fit = (question.fit ?? '').trim();
    final base = Theme.of(context).textTheme.bodyMedium;
    final labelStyle = base?.copyWith(
      fontSize: 16,
      height: 1.4,
      fontWeight: FontWeight.w700,
      color: AppStyles.primary900,
    );
    final questionStyle = base?.copyWith(
      fontSize: 16,
      height: 1.4,
      color: Colors.black,
      fontWeight: FontWeight.w600,
    );
    final bodyStyle = base?.copyWith(
      fontSize: 16,
      height: 1.4,
      color: Colors.black,
    );

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8.0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Línea inicial con estado + pregunta
          Text(
            'Pregunta $index: $statusText',
            style: labelStyle,
            textAlign: TextAlign.justify,
          ),
          const SizedBox(height: 6),
          Text(
            question.question,
            style: questionStyle,
            textAlign: TextAlign.justify,
          ),
          const SizedBox(height: 8),
          RichText(
            textAlign: TextAlign.justify,
            text: TextSpan(
              children: [
                TextSpan(text: 'Respuesta: ', style: labelStyle),
                TextSpan(
                  text: '${userAns.isEmpty ? '—' : userAns}.',
                  style: bodyStyle,
                ),
              ],
            ),
          ),
          if (fit.isNotEmpty) ...[
            const SizedBox(height: 8),
            RichText(
              textAlign: TextAlign.justify,
              text: TextSpan(
                children: [
                  TextSpan(text: 'Retroalimentación: ', style: labelStyle),
                  TextSpan(text: fit, style: bodyStyle),
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }
}
