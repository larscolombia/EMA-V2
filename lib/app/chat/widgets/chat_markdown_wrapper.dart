import 'package:flutter/material.dart';
import 'package:flutter_streaming_text_markdown/flutter_streaming_text_markdown.dart';

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
  /// OPTIMIZADO: Maneja texto concatenado por SSE streaming donde headers
  /// y párrafos llegan pegados sin saltos de línea apropiados.
  String _processRawContent(String rawContent) {
    if (rawContent.trim().isEmpty) return rawContent;

    String processed = rawContent;

    // 1. Convert literal \n to actual line breaks
    processed = processed.replaceAll('\\n', '\n');

    // 2. CRÍTICO PARA SSE: Añadir saltos ANTES de headers pegados a texto
    // "texto## Header" → "texto\n\n## Header"
    // "palabra.## Header" → "palabra.\n\n## Header"
    processed = processed.replaceAllMapped(
      RegExp(r'([^\n])(#{1,6}\s+[^\n])', multiLine: true),
      (match) {
        final before = match.group(1)!;
        final header = match.group(2)!;
        // Si hay contenido antes del header (no es inicio de línea), añadir saltos
        if (before.trim().isNotEmpty) {
          return '$before\n\n$header';
        }
        return '$before$header';
      },
    );

    // 3. Asegurar salto de línea DESPUÉS de headers si no existe
    // "## Header\nTexto" → "## Header\n\nTexto" (para separar header de contenido)
    processed = processed.replaceAllMapped(
      RegExp(r'(#{1,6}\s+[^\n]+)\n([^\n#])', multiLine: true),
      (match) {
        final header = match.group(1)!;
        final nextChar = match.group(2)!;
        return '$header\n\n$nextChar';
      },
    );

    // 4. Normalizar múltiples saltos a máximo 2 (Markdown paragraph spacing)
    processed = processed.replaceAll(RegExp(r'\n{3,}'), '\n\n');

    // 5. Limpiar artefactos de streaming: ## sueltos, espacios extras
    // "., (PMID)" → ". (PMID)" ; "(PubMed) (PubMed)" → "(PubMed)"
    processed = processed.replaceAll(RegExp(r'\.,\s*'), '. ');
    processed = processed.replaceAll(
      RegExp(r'\(PubMed\)\s*\(PubMed\)'),
      '(PubMed)',
    );

    // Limpiar ## sueltos al final de líneas o pegados a palabras
    // "Fuentes ###" → "Fuentes" ; ".## F" → "."
    processed = processed.replaceAll(RegExp(r'\.##\s*[A-Z]'), '.');
    processed = processed.replaceAll(RegExp(r'#{2,}\s*$', multiLine: true), '');

    // 6. Limpiar placeholders malformados legacy (compatibilidad)
    processed = processed.replaceAll(RegExp(r'\$1eferencias'), 'Referencias');
    processed = processed.replaceAll(RegExp(r'\$1esumen'), 'Resumen');
    processed = processed.replaceAll(RegExp(r'\$[12]\s*'), '');

    // 7. Normalizar espacios: múltiples espacios → uno solo (excepto saltos de línea)
    processed = processed.replaceAllMapped(
      RegExp(r'([^\n])\s{2,}([^\n])', multiLine: true),
      (match) => '${match.group(1)} ${match.group(2)}',
    );

    // 8. Convertir brackets especiales a normales
    processed = processed.replaceAll('【', '[');
    processed = processed.replaceAll('】', ']');

    // 9. Procesar placeholders si se proveen reemplazos
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
    // SOLUCIÓN: La librería flutter_streaming_text_markdown soporta headers Markdown nativamente.
    // NO necesitamos degradar headers (## → ###), solo asegurar formato correcto.
    // El backend ahora envía headers completos sin cortar.
    // IMPORTANTE: Usar .instant() para mostrar texto completo inmediatamente (sin animación).
    // El streaming ya se maneja en el controller (token por token del SSE).

    Widget content = Padding(
      padding: const EdgeInsets.symmetric(vertical: 4.0),
      child: StreamingTextMarkdown.instant(
        text: _processedText,
        markdownEnabled: true,
        theme: StreamingTextTheme(
          textStyle: widget.style.copyWith(
            height: 1.7,
            fontSize: widget.style.fontSize ?? 15,
          ),
          defaultPadding: const EdgeInsets.all(0),
        ),
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
