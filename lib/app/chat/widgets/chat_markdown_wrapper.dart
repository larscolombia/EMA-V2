import 'package:flutter/material.dart';
import 'package:gpt_markdown/gpt_markdown.dart';

/// A wrapper for GptMarkdown that processes raw medical content into well-formatted
/// Markdown with proper styling for titles, references, and clinical content.
///
/// This widget is optimized to handle the new backend format that includes:
/// - Direct medical responses without preambles
/// - Source indicators: *(Fuente: Base de conocimientos interna)*, *(Fuente: PubMed)*
/// - Clean **Referencias:** sections with proper formatting
/// - No duplicate references or malformed placeholders
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
  String _processRawContent(String rawContent) {
    if (rawContent.trim().isEmpty) return rawContent;

    String processed = rawContent;

    // 1. Convert literal \n to actual line breaks
    processed = processed.replaceAll('\\n', '\n');

    // 2. Handle new backend format source indicators (keep as is, they're already clean)
    // *(Fuente: Base de conocimientos interna)*, *(Fuente: PubMed)*, etc.
    // No processing needed - backend now sends clean format

    // 3. Handle new **Referencias:** format (already clean from backend)
    // No processing needed - backend sends properly formatted **Referencias:**

    // 4. Legacy cleanup (for any old content still in cache)
    processed = processed.replaceAll(RegExp(r'\$1eferencias'), 'Referencias');
    processed = processed.replaceAll(RegExp(r'\$1esumen'), 'Resumen');
    processed = processed.replaceAll(RegExp(r'\$2oi:'), 'doi:');
    processed = processed.replaceAll(RegExp(r'\$2ururo'), 'eururo');

    // 5. Clean up any remaining malformed placeholders (legacy support)
    processed = processed.replaceAll(
      RegExp(r'\$1[^a-zA-Z]'),
      ' ',
    ); // $1 followed by non-letter
    processed = processed.replaceAll(
      RegExp(r'\$2[^a-zA-Z]'),
      ' ',
    ); // $2 followed by non-letter
    processed = processed.replaceAll(
      RegExp(r'\$1$'),
      '',
    ); // $1 at end of string
    processed = processed.replaceAll(
      RegExp(r'\$2$'),
      '',
    ); // $2 at end of string

    // 6. Handle any remaining legacy PMID formats (backend should not send these anymore)
    processed = processed.replaceAll(
      RegExp(r'【PMID:\s*\$1】'),
      '',
    ); // Remove malformed PMID
    processed = processed.replaceAll(
      RegExp(r'\[PMID:\s*\$1\]'),
      '',
    ); // Remove malformed PMID
    processed = processed.replaceAll(
      RegExp(r'PMID:\s*\$1'),
      '',
    ); // Remove PMID with placeholder

    // 7. Ensure proper spacing around headers (minimal processing since backend is cleaner)
    processed = processed.replaceAll(
      RegExp(r'^(#{1,6})\s*(.+)$', multiLine: true),
      r'$1 $2',
    );

    // 8. Clean up excessive whitespace but preserve intentional breaks
    processed = processed.replaceAll(RegExp(r'\n{4,}'), '\n\n\n');
    processed = processed.replaceAll(RegExp(r'[ \t]+'), ' ');
    processed = processed.replaceAll(
      RegExp(r' +'),
      ' ',
    ); // Multiple spaces to single space

    // 9. Process placeholders if replacements are provided
    if (widget.placeholderReplacements != null) {
      widget.placeholderReplacements!.forEach((placeholder, replacement) {
        // Handle both $1(word) and $word formats
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
    final theme = Theme.of(context);

    // Custom theme for medical content with purple headers
    final customTheme = theme.copyWith(
      textTheme: theme.textTheme.copyWith(
        // H1 - Main titles
        displayLarge: theme.textTheme.displayLarge?.copyWith(
          color: const Color(0xFF6B46C1), // Purple-600
          fontSize: 28,
          fontWeight: FontWeight.bold,
          height: 1.3,
        ),
        // H2 - Section titles (Referencias, etc.)
        displayMedium: theme.textTheme.displayMedium?.copyWith(
          color: const Color(0xFF7C3AED), // Purple-500
          fontSize: 24,
          fontWeight: FontWeight.bold,
          height: 1.4,
        ),
        // H3 - Subsection titles
        displaySmall: theme.textTheme.displaySmall?.copyWith(
          color: const Color(0xFF8B5CF6), // Purple-400
          fontSize: 20,
          fontWeight: FontWeight.w600,
          height: 1.4,
        ),
        // H4 - Minor titles
        headlineLarge: theme.textTheme.headlineLarge?.copyWith(
          color: const Color(0xFFA855F7), // Purple-300
          fontSize: 18,
          fontWeight: FontWeight.w600,
          height: 1.4,
        ),
        // H5 - Small titles
        headlineMedium: theme.textTheme.headlineMedium?.copyWith(
          color: const Color(0xFFB794F6), // Purple-300 lighter
          fontSize: 16,
          fontWeight: FontWeight.w500,
          height: 1.4,
        ),
        // H6 - Minimal titles
        headlineSmall: theme.textTheme.headlineSmall?.copyWith(
          color: const Color(0xFFC4B5FD), // Purple-200
          fontSize: 14,
          fontWeight: FontWeight.w500,
          height: 1.4,
        ),
        // Body text
        bodyLarge: widget.style.copyWith(height: 1.6, fontSize: 16),
        bodyMedium: widget.style.copyWith(height: 1.6, fontSize: 14),
      ),
    );

    // Simple content without nested scrolls for chat messages
    Widget content = Theme(
      data: customTheme,
      child: DefaultTextStyle.merge(
        style: widget.style.copyWith(height: 1.6),
        textAlign: TextAlign.justify,
        child: GptMarkdown(_processedText),
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
