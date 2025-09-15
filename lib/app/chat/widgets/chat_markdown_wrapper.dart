import 'package:flutter/material.dart';
import 'package:gpt_markdown/gpt_markdown.dart';

/// A wrapper for GptMarkdown that handles layout and ensures proper rendering
/// of structured content like tables, code blocks, and long responses.
class ChatMarkdownWrapper extends StatefulWidget {
  final String text;
  final TextStyle style;

  const ChatMarkdownWrapper({
    super.key,
    required this.text,
    required this.style,
  });

  @override
  State<ChatMarkdownWrapper> createState() => _ChatMarkdownWrapperState();
}

class _ChatMarkdownWrapperState extends State<ChatMarkdownWrapper> {
  late String _processedText;

  @override
  void initState() {
    super.initState();
    _checkContentLength();
  }

  void _checkContentLength() {
    // Check if content is potentially complex based on markers
    final text = widget.text;

    // Mantener heurística pero sin guardar flags locales
    final _ =
        text.length > 1000 ||
        text.contains('| ---') || // Markdown tables
        text.contains('```') || // Code blocks
        text.contains('**Resumen estructurado**') || // Structured content
        text.split('\n').length > 20; // Many lines

    _processedText = _hardWrapLongTokens(text);
  }

  // Inserta saltos suaves en tokens muy largos sin espacios para permitir wrap
  String _hardWrapLongTokens(String input) {
    return input.replaceAllMapped(RegExp(r'(\S{40,})'), (m) {
      final s = m.group(0)!;
      final buf = StringBuffer();
      for (int i = 0; i < s.length; i++) {
        buf.write(s[i]);
        if ((i + 1) % 40 == 0) buf.write('\u200B'); // zero‑width space
      }
      return buf.toString();
    });
  }

  @override
  Widget build(BuildContext context) {
    Widget content = GptMarkdown(_processedText);

    if (_processedText.contains('| ---') || _processedText.contains('```')) {
      content = SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: ConstrainedBox(
          constraints: const BoxConstraints(minWidth: 0),
          child: content,
        ),
      );
    }
    // Forzar color y estilo por defecto (blanco) sobre todo el markdown
    final themed = Theme(
      data: Theme.of(context).copyWith(
        textTheme: Theme.of(context).textTheme.apply(
          bodyColor: widget.style.color,
          displayColor: widget.style.color,
        ),
      ),
      child: DefaultTextStyle.merge(style: widget.style, child: content),
    );
    return SelectionArea(child: themed);
  }
}
