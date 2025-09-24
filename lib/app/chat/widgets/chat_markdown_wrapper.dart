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

    // 2. Clean up duplicate reference sections and normalize structure
    processed = _consolidateReferences(processed);

    // 3. Format lists properly (convert numbered lists to markdown)
    processed = _formatLists(processed);

    // 3.1. Ensure proper spacing between numbered list items
    processed = processed.replaceAllMapped(
      RegExp(r'(\d+\.\s+\*\*[^*]+\*\*)\s*(\d+\.\s+\*\*)', multiLine: true),
      (match) => '${match.group(1)}\n\n${match.group(2)}',
    );

    // 3.2. Fix any numbered lists that are still stuck together
    processed = processed.replaceAllMapped(
      RegExp(r'(\w+\.\s*)(\d+\.\s+)', multiLine: true),
      (match) => '${match.group(1)}\n\n${match.group(2)}',
    );

    // 4. Handle source indicators - clean format
    processed = processed.replaceAllMapped(
      RegExp(r'\*\(Fuente:\s*([^)]+)\)\*'),
      (match) => '\n\n*Fuente: ${match.group(1)}*\n',
    );

    // 5. Legacy cleanup (for any old content still in cache)
    processed = processed.replaceAll(RegExp(r'\$1eferencias'), 'Referencias');
    processed = processed.replaceAll(RegExp(r'\$1esumen'), 'Resumen');
    processed = processed.replaceAll(RegExp(r'\$2oi:'), 'doi:');
    processed = processed.replaceAll(RegExp(r'\$2ururo'), 'eururo');

    // 6. Clean up any remaining malformed placeholders (legacy support)
    // Remove isolated $1 $2 patterns (most common case)
    processed = processed.replaceAll(RegExp(r'\$1\s+\$2\s*'), '');

    // Remove $1 or $2 at the end of lines or text
    processed = processed.replaceAll(
      RegExp(r'\s*\$[12]\s*$', multiLine: true),
      '',
    );

    // Remove $1 or $2 followed by non-alphabetic characters (but preserve if part of word)
    processed = processed.replaceAll(RegExp(r'\$1(?![a-zA-Z])'), '');
    processed = processed.replaceAll(RegExp(r'\$2(?![a-zA-Z])'), '');

    // Clean up any resulting double spaces
    processed = processed.replaceAll(RegExp(r'\s{2,}'), ' ');

    // 7. Handle malformed PMID references
    processed = processed.replaceAllMapped(
      RegExp(r'【PMID:\s*(\d+)】'),
      (match) => '[PMID: ${match.group(1)}]',
    );
    processed = processed.replaceAll(RegExp(r'【PMID:\s*\$1】'), '');
    processed = processed.replaceAll(RegExp(r'\[PMID:\s*\$1\]'), '');
    processed = processed.replaceAll(RegExp(r'PMID:\s*\$1'), '');

    // 8. Fix missing spaces after ## in headers
    processed = processed.replaceAll(RegExp(r'##([^\s#])'), '## \$1');

    // 8.1. Also ensure proper line breaks before headers
    processed = processed.replaceAll(RegExp(r'([^\n])(\n## )'), '\$1\n\n\$2');

    // 8.1. Ensure proper spacing around all headers
    processed = processed.replaceAllMapped(
      RegExp(r'^(#{1,6})\s*(.+)$', multiLine: true),
      (match) => '${match.group(1)} ${match.group(2)}',
    );

    // 9. Clean up excessive whitespace but preserve intentional breaks
    processed = processed.replaceAll(RegExp(r'\n{4,}'), '\n\n\n');
    processed = processed.replaceAll(RegExp(r'[ \t]+'), ' ');
    processed = processed.replaceAll(RegExp(r' +'), ' ');

    // 10. Process placeholders if replacements are provided
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

  /// Consolidates duplicate reference sections and cleans up formatting
  String _consolidateReferences(String content) {
    // First, remove all embedded reference sections in the middle of content
    content = _removeEmbeddedReferenceSections(content);

    // Find all reference sections
    final referenceSections = <String>[];
    final pmidReferences = <String, String>{};

    // Extract Referencias: sections (both with and without **)
    final referencePattern = RegExp(
      r'(?:Referencias?:|(?:\*\*)?Referencias?(?:\*\*)?:?)\s*\n((?:(?:-\s*)?[^\n]+(?:\n|$))*)',
      multiLine: true,
      caseSensitive: false,
    );

    final matches = referencePattern.allMatches(content);
    for (final match in matches) {
      if (match.group(1) != null) {
        referenceSections.add(match.group(1)!.trim());
      }
    }

    // Extract individual references and deduplicate by PMID
    for (final section in referenceSections) {
      final lines = section.split('\n');
      for (final line in lines) {
        final cleanLine = line.replaceAll(RegExp(r'^-\s*-?\s*'), '').trim();
        if (cleanLine.isNotEmpty &&
            !cleanLine.startsWith('*') &&
            cleanLine.length > 10) {
          // Extract PMID if present
          final pmidMatch = RegExp(r'PMID:\s*(\d+)').firstMatch(cleanLine);
          if (pmidMatch != null) {
            final pmid = pmidMatch.group(1)!;
            // Keep the most complete reference for each PMID
            if (!pmidReferences.containsKey(pmid) ||
                cleanLine.length > pmidReferences[pmid]!.length) {
              pmidReferences[pmid] = cleanLine;
            }
          } else {
            // Non-PMID references - keep unique ones but avoid very short lines
            final key = cleanLine.substring(
              0,
              cleanLine.length < 50 ? cleanLine.length : 50,
            );
            if (!pmidReferences.containsKey(key)) {
              pmidReferences[key] = cleanLine;
            }
          }
        }
      }
    }

    // Remove all existing reference sections
    String cleaned = content.replaceAll(referencePattern, '');

    // Detect trailing question to preserve it as the last line
    String? trailingQuestion;
    final tqMatch = RegExp(
      r'(.*?)(\s*)(¿[^?]*\?)\s*$',
      dotAll: true,
    ).firstMatch(cleaned);
    if (tqMatch != null) {
      cleaned = tqMatch.group(1)!.trimRight();
      trailingQuestion = tqMatch.group(3)!.trim();
    }

    // Add consolidated references section if we have any
    if (pmidReferences.isNotEmpty) {
      final sortedRefs = pmidReferences.values.toList()..sort();
      final buffer = StringBuffer();
      buffer.write(cleaned.trimRight());
      buffer.write('\n\n**Referencias:**\n');
      for (final ref in sortedRefs) {
        buffer.write('- ');
        buffer.write(ref);
        buffer.write('\n');
      }
      if (trailingQuestion != null && trailingQuestion.isNotEmpty) {
        buffer.write('\n');
        buffer.write(trailingQuestion);
      }
      return buffer.toString().trim();
    }

    // No references to add; if there was a trailing question, re-append it
    if (trailingQuestion != null && trailingQuestion.isNotEmpty) {
      cleaned = cleaned.trimRight() + '\n\n' + trailingQuestion;
    }

    return cleaned.trim();
  }

  /// Removes embedded reference sections that appear in the middle of content
  String _removeEmbeddedReferenceSections(String content) {
    // Remove reference sections that appear before the final **(Referencias:** section
    final parts = content.split('**Referencias:**');
    if (parts.length > 1) {
      // If there's a final Referencias section, clean the content before it
      final mainContent = parts[0];
      final finalRefs = parts.sublist(1).join('**Referencias:**');

      // Remove any Referencias: sections from main content
      final cleanedMain = mainContent.replaceAll(
        RegExp(
          r'Referencias?\s*:?\s*\n(?:\s*-[^\n]*\n?)*',
          multiLine: true,
          caseSensitive: false,
        ),
        '',
      );

      return '$cleanedMain\n\n**Referencias:**$finalRefs';
    }

    // If no final Referencias section, just remove embedded ones
    return content.replaceAll(
      RegExp(
        r'Referencias?\s*:?\s*\n(?:\s*-[^\n]*\n?)*',
        multiLine: true,
        caseSensitive: false,
      ),
      '',
    );
  }

  /// Formats numbered lists into proper markdown
  String _formatLists(String content) {
    // First, ensure proper spacing around numbered lists
    // Convert numbered lists (1. 2. 3.) to proper markdown with line breaks
    String formatted = content.replaceAllMapped(
      RegExp(
        r'(\d+)\.\s*([^0-9\n]*?)(?=\d+\.\s|\n\n|\*\(Fuente|\*\*Referencias|\$)',
        multiLine: true,
        dotAll: true,
      ),
      (match) {
        final number = match.group(1)!;
        final text = match.group(2)!.trim();
        // Ensure each numbered item is on its own line with proper spacing
        return '$number. **${text.replaceAll(RegExp(r'\s+'), ' ')}**\n\n';
      },
    );

    // Handle the last numbered item that might not have a following pattern
    formatted = formatted.replaceAllMapped(
      RegExp(r'^(\d+)\.\s+(.+)$', multiLine: true),
      (match) {
        final number = match.group(1)!;
        final text = match.group(2)!.trim();
        // Only format if it hasn't been formatted already
        if (!text.startsWith('**')) {
          return '$number. **$text**';
        }
        return match.group(0)!;
      },
    );

    // Clean up any triple line breaks that might have been created
    formatted = formatted.replaceAll(RegExp(r'\n{3,}'), '\n\n');

    return formatted;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    // Custom theme for medical content with white/light colors for dark chat background
    final customTheme = theme.copyWith(
      textTheme: theme.textTheme.copyWith(
        // H1 - Main titles
        displayLarge: widget.style.copyWith(
          fontSize: 18,
          fontWeight: FontWeight.bold,
          height: 1.3,
        ),
        // H2 - Section titles (Evidencia usada, Fuentes, etc.)
        displayMedium: widget.style.copyWith(
          fontSize: 16,
          fontWeight: FontWeight.bold,
          height: 1.4,
        ),
        // H3 - Subsection titles
        displaySmall: widget.style.copyWith(
          fontSize: 15,
          fontWeight: FontWeight.w600,
          height: 1.4,
        ),
        // H4 - Minor titles
        headlineLarge: widget.style.copyWith(
          fontSize: 15,
          fontWeight: FontWeight.w600,
          height: 1.4,
        ),
        // H5 - Small titles
        headlineMedium: widget.style.copyWith(
          fontSize: 14,
          fontWeight: FontWeight.w500,
          height: 1.4,
        ),
        // H6 - Minimal titles
        headlineSmall: widget.style.copyWith(
          fontSize: 14,
          fontWeight: FontWeight.w500,
          height: 1.4,
        ),
        // Body text - use the same style passed from ChatMessageAi
        bodyLarge: widget.style.copyWith(height: 1.5),
        bodyMedium: widget.style.copyWith(height: 1.5),
      ),
    );

    // Simple content without nested scrolls for chat messages
    Widget content = Theme(
      data: customTheme,
      child: DefaultTextStyle.merge(
        style: widget.style.copyWith(height: 1.5),
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
