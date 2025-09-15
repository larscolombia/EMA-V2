import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';

class FullFeedbackAnimated extends StatefulWidget {
  final String fitGlobal;
  final List<QuestionResponseModel> questions;
  final bool animate;
  const FullFeedbackAnimated({
    super.key,
    required this.fitGlobal,
    required this.questions,
    this.animate = true,
  });

  @override
  State<FullFeedbackAnimated> createState() => _FullFeedbackAnimatedState();
}

class _FullFeedbackAnimatedState extends State<FullFeedbackAnimated> {
  late final _Parsed parsed;
  // Los segmentos y conteos se construyen en build para usar Theme.of(context) sin depender en initState.
  bool _showAll = false; // permite saltar animación al tocar

  @override
  void initState() {
    super.initState();
    parsed = _parseFitGlobal(widget.fitGlobal);
  }

  @override
  Widget build(BuildContext context) {
    final segments = _buildSegments(context);
    final totalChars = segments.fold<int>(0, (sum, s) => sum + s.text.length);
    Widget content;
    if (_showAll || !widget.animate) {
      content = _buildRichText(totalChars, segments); // full render
    } else {
      final seconds = (totalChars / 55).clamp(3, 25); // velocidad
      content = TweenAnimationBuilder<int>(
        tween: IntTween(begin: 0, end: totalChars),
        duration: Duration(seconds: seconds.toInt()),
        curve: Curves.linear,
        builder: (context, value, child) => _buildRichText(value, segments),
      );
    }
    return GestureDetector(
      behavior: HitTestBehavior.translucent,
      onTap: () {
        if (!_showAll) {
          setState(() => _showAll = true);
        }
      },
      child: content,
    );
  }

  Widget _buildRichText(int visibleChars, List<_Segment> segments) {
    final spans = <TextSpan>[];
    var remaining = visibleChars;
    for (final seg in segments) {
      if (remaining <= 0) break;
      if (remaining >= seg.text.length) {
        spans.add(TextSpan(text: seg.text, style: seg.style));
        remaining -= seg.text.length;
      } else {
        spans.add(
          TextSpan(text: seg.text.substring(0, remaining), style: seg.style),
        );
        remaining = 0;
      }
    }
    return RichText(
      textAlign: TextAlign.justify,
      text: TextSpan(children: spans),
    );
  }

  List<_Segment> _buildSegments(BuildContext context) {
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

    final segs = <_Segment>[];
    void add(String text, TextStyle? style) {
      if (text.isEmpty) return;
      segs.add(_Segment(text: text, style: style));
    }

    // Puntaje y Calificación
    if (parsed.scoreBlock.isNotEmpty) {
      add('Puntaje y Calificación:\n', labelStyle);
      add(parsed.scoreBlock.trim() + '\n\n', bodyStyle);
    }
    // Retroalimentación
    if (parsed.feedbackBlock.isNotEmpty) {
      add('Retroalimentación:\n', labelStyle);
      add(parsed.feedbackBlock.trim() + '\n\n', bodyStyle);
    }
    // Preguntas
    for (var i = 0; i < widget.questions.length; i++) {
      final q = widget.questions[i];
      final status = (q.isCorrect == true) ? 'Correcta' : 'Incorrecta';
      add('Pregunta ${i + 1}: $status\n', labelStyle);
      add(q.question.trim() + '\n\n', questionStyle);
      final userAns = (q.answerdString).trim();
      add('Respuesta: ', labelStyle);
      add((userAns.isEmpty ? '—' : userAns) + '.\n', bodyStyle);
      if ((q.fit ?? '').trim().isNotEmpty) {
        add('Retroalimentación: ', labelStyle);
        add(q.fit!.trim() + '\n', bodyStyle);
      }
      add('\n', bodyStyle);
    }
    // Referencias al final
    if (parsed.references.isNotEmpty) {
      add('Referencias:\n', labelStyle);
      add(parsed.references.trim() + '\n', bodyStyle);
    }
    return segs;
  }

  _Parsed _parseFitGlobal(String s) {
    final norm = s.replaceAll('\r\n', '\n');
    final parts = norm.split('\n\n');
    String score = '';
    // feedback se construye en bufFeedback; variable temporal innecesaria eliminada
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
    return _Parsed(
      scoreBlock: score,
      feedbackBlock: bufFeedback.toString(),
      references: references,
    );
  }
}

class _Segment {
  final String text;
  final TextStyle? style;
  _Segment({required this.text, required this.style});
}

class _Parsed {
  final String scoreBlock;
  final String feedbackBlock;
  final String references;
  _Parsed({
    required this.scoreBlock,
    required this.feedbackBlock,
    required this.references,
  });
}
