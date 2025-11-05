import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';

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

class _ChatMarkdownWrapperState extends State<ChatMarkdownWrapper> {
  late String _processedText;

  @override
  void initState() {
    super.initState();
    _processedText = _processRawContent(widget.text);
  }

  @override
  void didUpdateWidget(ChatMarkdownWrapper oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.text != oldWidget.text ||
        widget.placeholderReplacements != oldWidget.placeholderReplacements) {
      _processedText = _processRawContent(widget.text);
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
    // DEBUG: Contar saltos de línea reales
    final realNewlines = '\n'.allMatches(_processedText).length;
    print(
      '[ChatMarkdownWrapper] 🔍 Saltos de línea reales en texto: $realNewlines',
    );
    print('[ChatMarkdownWrapper] 🔍 Longitud total: ${_processedText.length}');

    // Renderizar con flutter_markdown para respetar headings y párrafos
    final theme = Theme.of(context);
    final baseMdStyle = MarkdownStyleSheet.fromTheme(theme).copyWith(
      p: widget.style.copyWith(height: 1.7),
      h1: widget.style.copyWith(
        fontSize: (widget.style.fontSize ?? 15) + 6,
        fontWeight: FontWeight.bold,
      ),
      h2: widget.style.copyWith(
        fontSize: (widget.style.fontSize ?? 15) + 4,
        fontWeight: FontWeight.bold,
      ),
      h3: widget.style.copyWith(
        fontSize: (widget.style.fontSize ?? 15) + 2,
        fontWeight: FontWeight.bold,
      ),
    );

    Widget content = Padding(
      padding: const EdgeInsets.symmetric(vertical: 4.0),
      child: MarkdownBody(
        data: _processedText,
        selectable: true,
        styleSheet: baseMdStyle,
      ),
    );

    // Only add horizontal scroll for very wide content like tables
    if (_processedText.contains('| ---') || _processedText.contains('```')) {
      content = SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: content,
      );
    }

    return SelectionArea(child: content);
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
