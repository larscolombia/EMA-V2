import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
// gpt_markdown is used via ChatMarkdownWrapper
import 'package:ema_educacion_medica_avanzada/app/chat/widgets/chat_markdown_wrapper.dart';

class FullFeedbackAnimated extends StatefulWidget {
  final String fitGlobal;
  final List<QuestionResponseModel> questions;
  final bool animate;
  // When true, render feedback and references using Markdown widgets (no typing animation).
  final bool renderMarkdown;
  const FullFeedbackAnimated({
    super.key,
    required this.fitGlobal,
    required this.questions,
    this.animate = true,
    this.renderMarkdown = false,
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
    if (widget.renderMarkdown) {
      // Prefer precise markdown rendering for headings/lists; disable typing animation.
      return _buildWidgetMode(context);
    }
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

  // Widget-based rendering with Markdown support (no animation)
  Widget _buildWidgetMode(BuildContext context) {
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
    final mdBaseStyle = TextStyle(
      height: 1.35,
      color: (bodyStyle?.color ?? Colors.black),
    ); // no fontSize to keep heading hierarchy

    final children = <Widget>[];
    void addSpacing([double h = 16]) => children.add(SizedBox(height: h));

    // Score block (como markdown para mostrar subsecciones con formato)
    if (parsed.scoreBlock.isNotEmpty) {
      final mdScore = _normalizeMarkdown(parsed.scoreBlock.trim());
      // CRÍTICO: Solo mostrar título "Puntaje y Calificacion:" si el contenido
      // NO comienza con un header Markdown (casos interactivos/quizzes)
      // Para casos analíticos que comienzan con "# Resumen Clínico", omitir el título
      final startsWithMarkdownHeader = RegExp(
        r'^\s{0,3}#{1,6}\s',
      ).hasMatch(mdScore);

      if (!startsWithMarkdownHeader) {
        children.add(
          Text(
            'Puntaje y Calificacion:',
            style: labelStyle,
            textAlign: TextAlign.justify,
          ),
        );
        children.add(const SizedBox(height: 8));
      }

      children.add(
        Theme(
          data: _markdownTheme(context, mdBaseStyle.color),
          child: ChatMarkdownWrapper(text: mdScore, style: mdBaseStyle),
        ),
      );
      addSpacing();
    }

    // Feedback (markdown)
    if (parsed.feedbackBlock.isNotEmpty) {
      final md = _normalizeMarkdown(parsed.feedbackBlock.trim());
      final startsWithHeading = RegExp(r'^\s{0,3}#{1,6}\s').hasMatch(md);
      if (!startsWithHeading) {
        children.add(
          Text(
            'Retroalimentación:',
            style: labelStyle,
            textAlign: TextAlign.justify,
          ),
        );
        children.add(const SizedBox(height: 8));
      }
      children.add(
        Theme(
          data: _markdownTheme(context, mdBaseStyle.color),
          child: ChatMarkdownWrapper(text: md, style: mdBaseStyle),
        ),
      );
      addSpacing();
    }

    // Questions
    for (var i = 0; i < widget.questions.length; i++) {
      final q = widget.questions[i];
      final status = (q.isCorrect == true) ? 'Correcta' : 'Incorrecta';
      children.add(
        Text(
          'Pregunta ${i + 1}: $status',
          style: labelStyle,
          textAlign: TextAlign.justify,
        ),
      );
      children.add(const SizedBox(height: 6));
      children.add(
        SelectableText(
          q.question.trim(),
          style: questionStyle,
          textAlign: TextAlign.justify,
        ),
      );
      children.add(const SizedBox(height: 8));
      final userAns = (q.answerdString).trim();
      children.add(
        SelectableText(
          'Respuesta: ${userAns.isEmpty ? '—' : userAns}.',
          style: bodyStyle,
          textAlign: TextAlign.justify,
        ),
      );
      final fit = (q.fit ?? '').trim();
      if (fit.isNotEmpty) {
        children.add(const SizedBox(height: 8));
        final mdQ = _normalizeMarkdown(fit);
        final startsWithHeadingQ = RegExp(r'^\s{0,3}#{1,6}\s').hasMatch(mdQ);
        if (!startsWithHeadingQ) {
          children.add(
            Text(
              'Retroalimentación:',
              style: labelStyle,
              textAlign: TextAlign.justify,
            ),
          );
          children.add(const SizedBox(height: 6));
        }
        children.add(
          Theme(
            data: _markdownTheme(context, mdBaseStyle.color),
            child: ChatMarkdownWrapper(text: mdQ, style: mdBaseStyle),
          ),
        );
      }
      addSpacing();
    }

    // References (markdown) - Solo mostrar si hay contenido REAL (no solo encabezado vacío)
    final cleanRefs = parsed.references.trim();
    if (cleanRefs.isNotEmpty && cleanRefs.length > 10) {
      // Al menos 10 caracteres para evitar solo "Referencias:"
      final mdRef = _normalizeMarkdown(cleanRefs);
      final startsWithHeading = RegExp(r'^\s{0,3}#{1,6}\s').hasMatch(mdRef);
      if (!startsWithHeading) {
        children.add(
          Text('Referencias:', style: labelStyle, textAlign: TextAlign.justify),
        );
        children.add(const SizedBox(height: 8));
      }
      children.add(
        Theme(
          data: _markdownTheme(context, mdBaseStyle.color),
          child: ChatMarkdownWrapper(text: mdRef, style: mdBaseStyle),
        ),
      );
      addSpacing(8);
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: children,
    );
  }

  String _normalizeMarkdown(String s) {
    var text = s.replaceAll('\r\n', '\n');

    // PASO 1: Limpiar # sueltos que puedan romper el markdown
    // Eliminar # aislados al final de líneas o párrafos
    text = text.replaceAll(RegExp(r'\s+#\s*$', multiLine: true), '');
    text = text.replaceAll(RegExp(r'\s+##\s*$', multiLine: true), '');
    text = text.replaceAll(RegExp(r'\.\s*#\s+'), '. ');

    // PASO 2: Normalizar headers markdown SOLO si NO están ya formateados correctamente
    // CRÍTICO: Para casos analíticos del backend, los headers ya vienen bien formateados
    // Solo normalizar headers mal formados (sin #, con :, pegados a texto anterior)
    final sectionPatterns = {
      'Resumen Clínico': 'Resumen Clínico',
      'Resumen clínico': 'Resumen Clínico',
      'Desempeño [Gg]lobal': 'Desempeño Global',
      'Áreas de mejor\\s*a': 'Áreas de Mejora',
      'Areas de mejora': 'Áreas de Mejora',
      'Recomendaciones [Aa]ccionable\\s*s?': 'Recomendaciones',
      'Recomendaciones': 'Recomendaciones',
      'Errores [Cc]rítico\\s*s?': 'Errores Críticos',
      'Errores [Cc]ritico\\s*s?': 'Errores Críticos',
      'Fortaleza\\s*s?': 'Fortalezas',
      'Puntuación': 'Puntuación',
      'Puntuacion': 'Puntuación',
      'Puntaje': 'Puntaje',
      'Referencias': 'Referencias',
      'Síntesis': 'Síntesis',
      'Sintesis': 'Síntesis',
    };

    for (final entry in sectionPatterns.entries) {
      final pattern = entry.key;
      final normalized = entry.value;

      // Patrón 1: Título SIN ## al inicio (casos legacy de quizzes)
      // NO tocar si ya tiene formato "## Título"
      text = text.replaceAllMapped(
        RegExp(
          '(?:^|\\n)\\s*($pattern)\\s*:',
          caseSensitive: false,
          multiLine: true,
        ),
        (m) {
          // Solo reemplazar si NO tiene ## antes
          final fullMatch = m.group(0)!;
          if (fullMatch.contains('##')) {
            return fullMatch; // Ya está bien formateado, no tocar
          }
          return '\n\n## $normalized\n\n';
        },
      );

      // Patrón 2: Título pegado después de punto (sin ##)
      text = text.replaceAllMapped(
        RegExp('([.!?])\\s*($pattern)\\s*:', caseSensitive: false),
        (m) {
          final fullMatch = m.group(0)!;
          if (fullMatch.contains('##')) {
            return fullMatch; // Ya está bien formateado
          }
          return '${m.group(1)}\n\n## $normalized\n\n';
        },
      );
    }

    // PASO 3: Asegurar espacio después de # en headers mal formados
    // Solo si el header NO tiene espacio después de ##
    text = text.replaceAllMapped(
      RegExp(r'^(\s*#{1,6})([^#\s])', multiLine: true),
      (m) => '${m.group(1)} ${m.group(2)}',
    );

    // PASO 4: Limpiar duplicados como "PuntuaciónPuntuación"
    text = text.replaceAllMapped(RegExp(r'(\w+)\1', caseSensitive: false), (m) {
      final word = m.group(1)!;
      // Solo limpiar si es una palabra larga (título duplicado)
      return word.length > 5 ? word : m.group(0)!;
    });

    // PASO 5: Separar items numerados largos
    text = text.replaceAllMapped(
      RegExp(r'([.!?])\s*(\d+[\).])\s+'),
      (m) => '${m.group(1)}\n\n${m.group(2)} ',
    );

    // PASO 6: Limpiar excesos
    // Múltiples saltos de línea
    text = text.replaceAll(RegExp(r'\n{3,}'), '\n\n');
    // Saltos al inicio
    text = text.replaceAll(RegExp(r'^\n+'), '');
    // Espacios al final de líneas
    text = text.replaceAll(RegExp(r' +$', multiLine: true), '');

    // PASO 7: Convertir listas numeradas 1) a 1.
    final lines = text.split('\n');
    final out = <String>[];
    final reNumParen = RegExp(r'^\s*(\d+)\)\s+');
    for (final line in lines) {
      var processedLine = line;
      final m = reNumParen.firstMatch(line);
      if (m != null) {
        final num = m.group(1);
        processedLine = line.replaceFirst(reNumParen, '$num. ');
      }
      out.add(processedLine);
    }

    return out.join('\n').trim();
  }

  ThemeData _markdownTheme(BuildContext context, Color? baseColor) {
    final theme = Theme.of(context);
    // Reduce default heading sizes and keep contrast; body text stays around 16 via DefaultTextStyle
    final tt = theme.textTheme;
    return theme.copyWith(
      textTheme: tt.copyWith(
        headlineSmall: tt.headlineSmall?.copyWith(
          fontSize: 18,
          fontWeight: FontWeight.w700,
          color: baseColor,
        ), // h2/h3 range
        titleLarge: tt.titleLarge?.copyWith(
          fontSize: 17,
          fontWeight: FontWeight.w700,
          color: baseColor,
        ),
        titleMedium: tt.titleMedium?.copyWith(
          fontSize: 16.5,
          fontWeight: FontWeight.w700,
          color: baseColor,
        ),
        titleSmall: tt.titleSmall?.copyWith(
          fontSize: 16,
          fontWeight: FontWeight.w700,
          color: baseColor,
        ),
        bodyLarge: tt.bodyLarge?.copyWith(fontSize: 16, color: baseColor),
        bodyMedium: tt.bodyMedium?.copyWith(fontSize: 16, color: baseColor),
        bodySmall: tt.bodySmall?.copyWith(
          fontSize: 15,
          color: baseColor?.withOpacity(0.9),
        ),
      ),
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
      add('Evaluación Final:\n', labelStyle);
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

    // Estrategia mejorada: buscar índices de secciones principales
    final lines = norm.split('\n');
    String score = '';
    String feedback = '';
    String references = '';

    int refStartIdx = -1;
    int scoreStartIdx = -1;
    int retroStartIdx = -1;

    // Primera pasada: identificar índices de inicio de cada sección
    for (int i = 0; i < lines.length; i++) {
      final trimmed = lines[i].trim();
      final lower = trimmed.toLowerCase();

      // Detectar sección de Referencias (formato backend: "## Referencias" o "Referencias:")
      if ((lower.startsWith('## referencias') ||
              lower.startsWith('referencias:')) &&
          refStartIdx == -1) {
        refStartIdx = i;
      }
      // Detectar inicio de evaluación/puntaje (múltiples formatos)
      else if ((lower.startsWith('puntaje y calificación') ||
              lower.startsWith('puntaje y calificacion') ||
              lower.startsWith('puntuación') ||
              lower.startsWith('puntuacion') ||
              lower.startsWith('## puntuación') ||
              lower.startsWith('## puntuacion') ||
              lower.startsWith('calificación') ||
              lower.startsWith('calificacion') ||
              lower.startsWith('resumen final:') || // Formato backend nuevo
              lower.startsWith('# resumen clínico') || // Casos analíticos
              lower.startsWith('# resumen clinico') ||
              lower.startsWith('puntaje:')) && // Línea de puntaje directo
          scoreStartIdx == -1) {
        scoreStartIdx = i;
      }
      // Detectar retroalimentación (opcional, solo en quizzes)
      else if ((lower.startsWith('retroalimentación') ||
              lower.startsWith('retroalimentacion')) &&
          retroStartIdx == -1) {
        retroStartIdx = i;
      }
    }

    // Segunda pasada: extraer contenido de cada sección
    // SCORE: captura TODO hasta referencias o retroalimentación
    // CRÍTICO: Para casos analíticos, si no hay scoreStartIdx detectado pero sí refStartIdx,
    // entonces TODO antes de referencias es el score (incluye # Resumen Clínico completo)
    if (scoreStartIdx >= 0) {
      final scoreLines = <String>[];
      for (int i = scoreStartIdx; i < lines.length; i++) {
        // Detener solo en secciones diferentes (Referencias, Retroalimentación)
        if (i == refStartIdx || i == retroStartIdx) break;
        scoreLines.add(lines[i]);
      }
      score =
          scoreLines
              .join('\n')
              .replaceFirst(
                RegExp(
                  r'^(puntaje y calificaci[oó]n|puntuaci[oó]n|calificaci[oó]n|resumen final)\s*:\s*',
                  caseSensitive: false,
                ),
                '',
              )
              .trim();
    } else if (refStartIdx >= 0) {
      // CASOS ANALÍTICOS: Sin scoreStartIdx detectado, score es TODO antes de referencias
      final scoreLines = lines.sublist(0, refStartIdx);
      score = scoreLines.join('\n').trim();
    } else if (scoreStartIdx == -1 && refStartIdx == -1) {
      // Sin ningún índice detectado: todo es score
      score = norm.trim();
    }

    if (retroStartIdx >= 0) {
      final feedbackLines = <String>[];
      for (int i = retroStartIdx; i < lines.length; i++) {
        if (i == refStartIdx ||
            (scoreStartIdx > retroStartIdx && i == scoreStartIdx))
          break;
        feedbackLines.add(lines[i]);
      }
      feedback =
          feedbackLines
              .join('\n')
              .replaceFirst(
                RegExp(r'^retroalimentaci[oó]n:\s*', caseSensitive: false),
                '',
              )
              .trim();
    }

    if (refStartIdx >= 0) {
      final refLines = <String>[];
      // CRÍTICO: Incluir el encabezado "## Referencias" en el contenido
      // para que ChatMarkdownWrapper lo renderice correctamente
      refLines.add(lines[refStartIdx]); // Agregar encabezado

      // Tomar TODAS las líneas después de "Referencias:" hasta el final del documento
      for (int i = refStartIdx + 1; i < lines.length; i++) {
        final trimmed = lines[i].trim();
        // NO detenerse en subsecciones de referencias (ej: "- Libro X")
        // Solo detener si encontramos otra sección PRINCIPAL que no sea parte de referencias
        if (trimmed.toLowerCase().startsWith('resumen final:') ||
            trimmed.toLowerCase().startsWith('retroalimentación:') ||
            trimmed.toLowerCase().startsWith('# ')) {
          break;
        }
        refLines.add(lines[i]);
      }
      references = refLines.join('\n').trim();
    }

    // Fallback: si no se detectaron secciones, usar lógica original por bloques
    if (score.isEmpty && feedback.isEmpty && references.isEmpty) {
      final parts = norm.split('\n\n');
      final bufFeedback = StringBuffer();
      for (final p in parts) {
        final t = p.trim();
        if (t.isEmpty) continue;
        final lt = t.toLowerCase();
        if (lt.startsWith('puntaje y calificación') ||
            lt.startsWith('puntaje y calificacion') ||
            lt.startsWith('puntuación') ||
            lt.startsWith('puntuacion') ||
            lt.startsWith('calificación') ||
            lt.startsWith('calificacion')) {
          score =
              t
                  .replaceFirst(
                    RegExp(
                      r'^(puntaje y calificaci[oó]n|puntuaci[oó]n|calificaci[oó]n)\s*:\s*',
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
    }

    return _Parsed(
      scoreBlock: score,
      feedbackBlock: feedback,
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
