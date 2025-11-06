import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:markdown/markdown.dart' as md;

/// Custom builder para párrafos justificados
class JustifiedParagraphBuilder extends MarkdownElementBuilder {
  final TextStyle? style;

  JustifiedParagraphBuilder({this.style});

  @override
  void visitElementBefore(md.Element element) {
    // No hacemos nada aquí
  }

  @override
  Widget? visitText(md.Text text, TextStyle? preferredStyle) {
    return RichText(
      textAlign: TextAlign.justify,
      text: TextSpan(text: text.text, style: preferredStyle ?? style),
    );
  }
}

/// A wrapper for StreamingTextMarkdown that processes raw medical content into well-formatted
/// Markdown with proper styling for titles, references, and clinical content.
///
/// This widget is optimized to handle the new backend format that includes:
/// - Direct medical responses without preambles
/// - Source indicators: *(Fuente: Base de conocimientos interna)*, *(Fuente: PubMed)*
/// - Clean **Referencias:** sections with proper formatting
/// - No duplicate references or malformed placeholders
/// - Streaming support with proper line breaks and formatting
class ChatMarkdownWrapper extends StatefulWidget {
  final String text;
  final TextStyle style;
  final Map<String, String>? placeholderReplacements;

  const ChatMarkdownWrapper({
    super.key,
    required this.text,
    required this.style,
    this.placeholderReplacements,
  });

  @override
  State<ChatMarkdownWrapper> createState() => _ChatMarkdownWrapperState();
}

class _ChatMarkdownWrapperState extends State<ChatMarkdownWrapper>
    with SingleTickerProviderStateMixin {
  // Set estático para persistir entre reconstrucciones del widget
  // Guarda hashes de textos que ya fueron animados
  static final Set<int> _animatedTexts = {};

  late String _processedText;
  String _displayedText = '';
  bool _isAnimating = true;
  bool _showCursor = true;
  int _currentIndex = 0;
  bool _hasLoggedOnce = false; // Para evitar logs repetidos

  // Timer para la animación de typing (velocidad moderada, natural)
  Duration _typingSpeed = const Duration(milliseconds: 12);

  @override
  void initState() {
    super.initState();
    _processedText = _processRawContent(widget.text);
    _startTypingAnimation();
    _startCursorBlink();
  }

  @override
  void didUpdateWidget(ChatMarkdownWrapper oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.text != oldWidget.text ||
        widget.placeholderReplacements != oldWidget.placeholderReplacements) {
      final newProcessed = _processRawContent(widget.text);

      // Si el texto cambió, reiniciar animación desde donde quedó
      if (newProcessed != _processedText) {
        _processedText = newProcessed;
        if (_isAnimating) {
          _startTypingAnimation();
        }
      }
    }
  }

  @override
  void dispose() {
    _isAnimating = false;
    super.dispose();
  }

  void _startTypingAnimation() {
    if (!mounted) return;

    // Usar hash del texto para identificar si ya fue animado
    final textHash = _processedText.hashCode;

    // Si ya se animó este texto, mostrar completo inmediatamente
    if (_animatedTexts.contains(textHash)) {
      setState(() {
        _displayedText = _processedText;
        _isAnimating = false;
      });
      return;
    }

    // Mostrar todo el texto inmediatamente si es muy corto
    if (_processedText.length < 50) {
      setState(() {
        _displayedText = _processedText;
        _isAnimating = false;
        _animatedTexts.add(textHash);
      });
      return;
    }

    _currentIndex = 0;
    _isAnimating = true;
    _animateNextChunk();
  }

  void _animateNextChunk() {
    if (!mounted || !_isAnimating) return;

    if (_currentIndex >= _processedText.length) {
      if (mounted) {
        setState(() {
          _isAnimating = false;
          _displayedText = _processedText;
          _animatedTexts.add(_processedText.hashCode); // Marcar que ya se animó
        });
      }
      return;
    }

    // Animar en chunks medianos para balance entre fluidez y velocidad
    final chunkSize = 3;
    final endIndex = (_currentIndex + chunkSize).clamp(
      0,
      _processedText.length,
    );

    Future.delayed(_typingSpeed, () {
      if (!mounted || !_isAnimating) return;

      // Usar WidgetsBinding para evitar rebuilds durante scroll
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted || !_isAnimating) return;

        setState(() {
          _currentIndex = endIndex;
          _displayedText = _processedText.substring(0, endIndex);
        });
      });

      _animateNextChunk();
    });
  }

  void _startCursorBlink() {
    Future.delayed(const Duration(milliseconds: 530), () {
      if (!mounted || !_isAnimating) return;

      // Solo hacer setState si realmente está animando
      if (_isAnimating && mounted) {
        setState(() {
          _showCursor = !_showCursor;
        });
        _startCursorBlink();
      }
    });
  }

  void _skipAnimation() {
    if (_isAnimating) {
      setState(() {
        _isAnimating = false;
        _displayedText = _processedText;
        _currentIndex = _processedText.length;
        _animatedTexts.add(_processedText.hashCode); // Marcar que ya se animó
      });
    }
  }

  /// Processes raw medical content into well-formatted Markdown
  /// SIMPLIFICADO: El backend (Go) ya envía formato correcto vía evento __JSON__
  /// Solo aplicamos limpieza ligera para casos edge y compatibilidad legacy.
  String _processRawContent(String rawContent) {
    if (rawContent.trim().isEmpty) return rawContent;

    String processed = rawContent;

    // DEBUG: Verificar texto recibido y contar saltos de línea reales
    final newlineCount = '\n'.allMatches(processed).length;
    final doubleNewlineCount = '\n\n'.allMatches(processed).length;
    final preview =
        processed.length > 200 ? processed.substring(0, 200) : processed;
    print(
      '[ChatMarkdownWrapper] 📝 Texto: ${processed.length} chars, $newlineCount \\n, $doubleNewlineCount \\n\\n',
    );
    print('[ChatMarkdownWrapper] Preview: ${preview.replaceAll('\n', '⏎')}');

    // IMPORTANTE: El backend YA normaliza el texto completo en normalizeMarkdownFull()
    // antes de enviarlo vía __JSON__. Aquí solo hacemos limpieza final mínima.

    // 1. Normalizar múltiples saltos a máximo 2 (Markdown paragraph spacing)
    processed = processed.replaceAll(RegExp(r'\n{3,}'), '\n\n');

    // 2. Limpiar artefactos legacy: duplicados de fuentes
    processed = processed.replaceAll(
      RegExp(r'\(PubMed\)\s*\(PubMed\)'),
      '(PubMed)',
    );

    // 3. Convertir brackets especiales a normales
    processed = processed.replaceAll('【', '[');
    processed = processed.replaceAll('】', ']');

    // 4. Limpiar trailing whitespace al final de líneas
    processed = processed.replaceAll(RegExp(r'[ \t]+\n'), '\n');

    // 5. Procesar placeholders si se proveen reemplazos (casos interactivos)
    if (widget.placeholderReplacements != null) {
      widget.placeholderReplacements!.forEach((placeholder, replacement) {
        processed = processed.replaceAll(
          '\$$placeholder($placeholder)',
          replacement,
        );
        processed = processed.replaceAll('\$$placeholder', replacement);
      });
    }

    return processed.trim();
  }

  @override
  Widget build(BuildContext context) {
    // DEBUG: Contar saltos de línea reales (solo una vez)
    if (!_hasLoggedOnce && _processedText.isNotEmpty) {
      _hasLoggedOnce = true;
      final realNewlines = '\n'.allMatches(_processedText).length;
      print(
        '[ChatMarkdownWrapper] 🔍 Saltos de línea reales en texto: $realNewlines',
      );
      print(
        '[ChatMarkdownWrapper] 🔍 Longitud total: ${_processedText.length}',
      );
    }

    // Renderizar con flutter_markdown para respetar headings y párrafos
    final theme = Theme.of(context);
    final baseFontSize = widget.style.fontSize ?? 15;

    final baseMdStyle = MarkdownStyleSheet.fromTheme(theme).copyWith(
      // Párrafos: texto justificado con buen espaciado
      p: widget.style.copyWith(height: 1.6, fontSize: baseFontSize),
      pPadding: const EdgeInsets.only(bottom: 12.0),

      // Headers: bold y con buen espaciado
      h1: widget.style.copyWith(
        fontSize: baseFontSize + 8,
        fontWeight: FontWeight.bold,
        height: 1.4,
      ),
      h1Padding: const EdgeInsets.only(top: 16.0, bottom: 12.0),

      h2: widget.style.copyWith(
        fontSize: baseFontSize + 6,
        fontWeight: FontWeight.bold,
        height: 1.3,
      ),
      h2Padding: const EdgeInsets.only(top: 14.0, bottom: 10.0),

      h3: widget.style.copyWith(
        fontSize: baseFontSize + 4,
        fontWeight: FontWeight.w600,
        height: 1.3,
      ),
      h3Padding: const EdgeInsets.only(top: 12.0, bottom: 8.0),

      // Listas: con buen espaciado
      listBullet: widget.style.copyWith(fontSize: baseFontSize),
      listIndent: 24.0,

      // Énfasis
      strong: widget.style.copyWith(fontWeight: FontWeight.bold),
      em: widget.style.copyWith(fontStyle: FontStyle.italic),

      textAlign: WrapAlignment.start,
    );

    // Usar texto animado o completo según el estado
    final displayText =
        _isAnimating
            ? (_displayedText + (_showCursor ? '▌' : ''))
            : _processedText;

    Widget content = Padding(
      padding: const EdgeInsets.symmetric(vertical: 4.0),
      child: MarkdownBody(
        data: displayText,
        selectable:
            !_isAnimating, // Solo seleccionable cuando termina la animación
        styleSheet: baseMdStyle,
        builders: {
          'p': JustifiedParagraphBuilder(
            style: widget.style.copyWith(height: 1.7),
          ),
        },
      ),
    );

    // Only add horizontal scroll for very wide content like tables
    if (_processedText.contains('| ---') || _processedText.contains('```')) {
      content = SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: content,
      );
    }

    // Envolver en GestureDetector para permitir saltar la animación
    // CRÍTICO: usar RepaintBoundary para evitar rebuilds que causan "vibración"
    // Stack con overlay transparente para capturar taps en toda el área
    return RepaintBoundary(
      child: Stack(
        children: [
          SelectionArea(child: content),
          // Overlay transparente que captura taps en toda el área de la burbuja
          if (_isAnimating)
            Positioned.fill(
              child: GestureDetector(
                behavior: HitTestBehavior.translucent,
                onTap: _skipAnimation,
                child: Container(color: Colors.transparent),
              ),
            ),
          // Indicador visual sutil cuando se puede saltar
          // IgnorePointer para que no interfiera con el overlay
          if (_isAnimating)
            Positioned(
              right: 8,
              top: 8,
              child: IgnorePointer(
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.surfaceVariant.withOpacity(0.7),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        Icons.touch_app,
                        size: 14,
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                      const SizedBox(width: 4),
                      Text(
                        'Tap para saltar',
                        style: TextStyle(
                          fontSize: 11,
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

/// A complete screen widget for displaying medical content with full markdown formatting.
///
/// Example usage:
/// ```dart
/// MarkdownScreen(
///   title: 'Caso Clínico',
///   rawContent: '''
///   # Evaluación Cardiovascular\\n\\n1. Resumen Ejecutivo\\nEl paciente presenta...\\n\\nReferencias:\\nPMID: 12345678
///   ''',
///   placeholderReplacements: {
///     '1': 'paciente Juan Pérez',
///     'autor': 'Dr. García',
///   },
/// )
/// ```
class MarkdownScreen extends StatelessWidget {
  final String title;
  final String rawContent;
  final Map<String, String>? placeholderReplacements;
  final Color? backgroundColor;

  const MarkdownScreen({
    super.key,
    required this.title,
    required this.rawContent,
    this.placeholderReplacements,
    this.backgroundColor,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      backgroundColor: backgroundColor ?? theme.scaffoldBackgroundColor,
      appBar: AppBar(
        title: Text(title),
        backgroundColor: theme.primaryColor,
        foregroundColor: Colors.white,
        elevation: 0,
      ),
      body: SafeArea(
        child: ChatMarkdownWrapper(
          text: rawContent,
          style: TextStyle(
            fontSize: 16,
            color: theme.textTheme.bodyLarge?.color ?? Colors.black87,
            height: 1.6,
          ),
          placeholderReplacements: placeholderReplacements,
        ),
      ),
    );
  }
}
