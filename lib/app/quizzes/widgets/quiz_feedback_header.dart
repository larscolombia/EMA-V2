import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';

class QuizFeedbackHeader extends StatefulWidget {
  final String fitGlobal;
  final bool animate;
  const QuizFeedbackHeader({
    super.key,
    required this.fitGlobal,
    this.animate = true,
  });

  @override
  State<QuizFeedbackHeader> createState() => _QuizFeedbackHeaderState();
}

class _QuizFeedbackHeaderState extends State<QuizFeedbackHeader> {
  late final _FitParsed parsed;

  @override
  void initState() {
    super.initState();
    parsed = _parseFitGlobal(widget.fitGlobal);
  }

  @override
  Widget build(BuildContext context) {
    final base = Theme.of(context).textTheme.bodyMedium;
    final purpleBold = base?.copyWith(
      fontSize: 16,
      height: 1.4,
      fontWeight: FontWeight.w700,
      color: AppStyles.primary900,
    );
    final body = base?.copyWith(fontSize: 16, height: 1.4);

    final scoreText = parsed.scoreBlock;
    final retroText = parsed.feedbackBlock;
    final fullAnim =
        scoreText +
        (scoreText.isNotEmpty && retroText.isNotEmpty ? '\n\n' : '') +
        retroText;

    if (!widget.animate) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (scoreText.isNotEmpty) ...[
            Text(
              'Puntaje y Calificación:',
              style: purpleBold,
              textAlign: TextAlign.justify,
            ),
            const SizedBox(height: 8),
            Text(scoreText, style: body, textAlign: TextAlign.justify),
            const SizedBox(height: 16),
          ],
          if (retroText.isNotEmpty) ...[
            Text(
              'Retroalimentación:',
              style: purpleBold,
              textAlign: TextAlign.justify,
            ),
            const SizedBox(height: 8),
            Text(retroText, style: body, textAlign: TextAlign.justify),
            const SizedBox(height: 16),
          ],
        ],
      );
    }

    final totalChars = fullAnim.length;
    final seconds = (totalChars / 40).clamp(2, 15); // velocidad aproximada

    return TweenAnimationBuilder<int>(
      tween: IntTween(begin: 0, end: totalChars),
      duration: Duration(seconds: seconds.toInt()),
      builder: (context, value, child) {
        final visible = fullAnim.substring(0, value);
        String visScore = '';
        String visRetro = '';
        if (scoreText.isNotEmpty) {
          if (value <= scoreText.length) {
            visScore = visible;
          } else {
            visScore = scoreText;
            final retroStart =
                scoreText.length +
                (scoreText.isNotEmpty && retroText.isNotEmpty ? 2 : 0);
            if (value > retroStart && retroText.isNotEmpty) {
              final retroChars = value - retroStart;
              if (retroChars > 0) {
                final safe = retroChars.clamp(0, retroText.length);
                visRetro = retroText.substring(0, safe);
              }
            }
          }
        } else {
          visRetro = visible;
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (scoreText.isNotEmpty) ...[
              Text(
                'Puntaje y Calificación:',
                style: purpleBold,
                textAlign: TextAlign.justify,
              ),
              const SizedBox(height: 8),
              Text(visScore, style: body, textAlign: TextAlign.justify),
              const SizedBox(height: 16),
            ],
            if (retroText.isNotEmpty) ...[
              Text(
                'Retroalimentación:',
                style: purpleBold,
                textAlign: TextAlign.justify,
              ),
              const SizedBox(height: 8),
              Text(visRetro, style: body, textAlign: TextAlign.justify),
              const SizedBox(height: 16),
            ],
          ],
        );
      },
    );
  }

  _FitParsed _parseFitGlobal(String s) {
    final norm = s.replaceAll('\r\n', '\n');
    final parts = norm.split('\n\n');
    String score = '';
    String feedback = '';
    String references = '';
    final bufFeedback = StringBuffer();
    for (final p in parts) {
      final t = p.trim();
      if (t.isEmpty) continue;
      final lt = t.toLowerCase();
      if (lt.startsWith('puntaje y calificación') ||
          lt.startsWith('puntaje y calificacion')) {
        score =
            t
                .replaceFirst(
                  RegExp(
                    r'^puntaje y calificaci[oó]n:\s*',
                    caseSensitive: false,
                  ),
                  '',
                )
                .trim();
      } else if (lt.startsWith('retroalimentación') ||
          lt.startsWith('retroalimentacion')) {
        final clean =
            t
                .replaceFirst(
                  RegExp(r'^retroalimentaci[oó]n:\s*', caseSensitive: false),
                  '',
                )
                .trim();
        if (bufFeedback.isNotEmpty) bufFeedback.writeln('\n');
        bufFeedback.write(clean);
      } else if (lt.startsWith('referencias')) {
        references =
            t
                .replaceFirst(
                  RegExp(r'^referencias:\s*', caseSensitive: false),
                  '',
                )
                .trim();
      } else {
        if (bufFeedback.isNotEmpty) bufFeedback.writeln('\n');
        bufFeedback.write(t);
      }
    }
    feedback = bufFeedback.toString();
    return _FitParsed(
      scoreBlock: score,
      feedbackBlock: feedback,
      references: references,
    );
  }
}

class _FitParsed {
  final String scoreBlock;
  final String feedbackBlock;
  final String references;
  _FitParsed({
    required this.scoreBlock,
    required this.feedbackBlock,
    required this.references,
  });
}
