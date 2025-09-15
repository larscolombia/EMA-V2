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
  bool _isExpanded = false;
  bool _isLongContent = false;
  late String _processedText;
  
  @override
  void initState() {
    super.initState();
    _checkContentLength();
  }
  
  void _checkContentLength() {
    // Check if content is potentially complex based on markers
    final text = widget.text;
    
    _isLongContent = text.length > 1000 || 
                     text.contains('| ---') || // Markdown tables
                     text.contains('```') ||  // Code blocks
                     text.contains('**Resumen estructurado**') || // Structured content
                     text.split('\n').length > 20; // Many lines
    
    // Always start expanded for better visibility
    _isExpanded = true;

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
    Widget md = GptMarkdown(_processedText);

    if (_processedText.contains('| ---') || _processedText.contains('```')) {
      md = SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: ConstrainedBox(
          constraints: const BoxConstraints(minWidth: 0),
          child: md,
        ),
      );
    }
    return SelectionArea(child: md);
  }
}
