import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'animations/slide_in_left.dart';
import 'chat_markdown_wrapper.dart';

class ChatMessageAi extends StatefulWidget {
  final ChatMessageModel message;

  const ChatMessageAi({super.key, required this.message});

  @override
  State<ChatMessageAi> createState() => _ChatMessageAiState();
}

class _ChatMessageAiState extends State<ChatMessageAi>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<double> _fadeAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      duration: const Duration(milliseconds: 800), // Duración de la animación
      vsync: this,
    );

    _fadeAnimation = CurvedAnimation(
      parent: _controller,
      curve: Curves.easeInOut,
    );

    Future.delayed(const Duration(milliseconds: 200), () {
      if (mounted) _controller.forward();
    });
  }

  List<String> _extractSources(String text) {
    final List<String> sources = [];

    // Buscar sección de fuentes
    final sourcesRegex = RegExp(
      r'## Fuentes:\s*\n(.*?)(?=\n\n|\n## |$)',
      dotAll: true,
    );
    final sourcesMatch = sourcesRegex.firstMatch(text);

    if (sourcesMatch != null) {
      final sourcesText = sourcesMatch.group(1) ?? '';
      // Extraer elementos de lista
      final listRegex = RegExp(r'^- (.+)$', multiLine: true);
      sources.addAll(
        listRegex
            .allMatches(sourcesText)
            .map((match) => match.group(1)!.trim()),
      );
    }

    // Si no hay fuentes estructuradas, buscar patrones alternativos
    if (sources.isEmpty) {
      // Buscar líneas de fuente al final
      final altSourceRegex = RegExp(
        r'Fuente:\s*(.+?)(?=\n|$)',
        multiLine: true,
      );
      sources.addAll(
        altSourceRegex.allMatches(text).map((match) => match.group(1)!.trim()),
      );
    }

    return sources;
  }

  String _getMainContent(String text) {
    if (text.trim().isEmpty) return text;

    String content = text;

    // El ChatMarkdownWrapper ya maneja la limpieza de placeholders

    // Remover las secciones de fuentes para mostrar solo el contenido principal
    final sourcesRegex = RegExp(r'\n## Fuentes:\s*\n.*', dotAll: true);
    content = content.replaceAll(sourcesRegex, '');

    // También remover patrones de fuente simples al final
    final altSourceRegex = RegExp(
      r'\n\*?\(?(Fuente:.*?)\)?\*?$',
      multiLine: true,
    );
    content = content.replaceAll(altSourceRegex, '');

    return content.trim();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final String timeStr = DateFormat('HH:mm').format(widget.message.createdAt);

    // Debug: imprimir la longitud del texto del mensaje
    print(
      '🔍 [ChatMessageAi] Mensaje recibido - Longitud: ${widget.message.text.length}',
    );
    print('🔍 [ChatMessageAi] Texto completo: "${widget.message.text}"');
    print(
      '🔍 [ChatMessageAi] Contenido principal: "${_getMainContent(widget.message.text)}"',
    );

    // Si no hay texto útil, no renderizar la burbuja para evitar espacios en blanco y overflows innecesarios
    if (widget.message.text.trim().isEmpty) {
      return const SizedBox.shrink();
    }

    return Align(
      alignment: Alignment.centerLeft,
      child: SlideInLeft(
        child: FadeTransition(
          opacity: _fadeAnimation,
          child: Container(
            margin: const EdgeInsets.only(
              top: 8,
              bottom: 8,
              right: 8, // Reducido el margen derecho
              left: 8, // Reducido el margen izquierdo
            ),
            // Usar todo el ancho disponible sin restricciones
            width: double.infinity,
            decoration: BoxDecoration(
              color: const Color.fromRGBO(58, 12, 140, 0.9),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Contenido principal de la respuesta
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: ChatMarkdownWrapper(
                    text: _getMainContent(widget.message.text),
                    style: const TextStyle(
                      fontSize: 15,
                      color: Colors.white,
                      height: 1.5,
                    ),
                  ),
                ),

                // Mostrar fuentes si están disponibles
                if (_extractSources(widget.message.text).isNotEmpty)
                  Container(
                    margin: const EdgeInsets.only(
                      left: 16,
                      right: 16,
                      bottom: 8,
                    ),
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: Colors.white.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Icon(Icons.source, color: Colors.white70, size: 16),
                            const SizedBox(width: 8),
                            Text(
                              'Fuentes',
                              style: TextStyle(
                                color: Colors.white70,
                                fontSize: 13,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 8),
                        ..._extractSources(widget.message.text).map(
                          (source) => Padding(
                            padding: const EdgeInsets.only(bottom: 4),
                            child: Row(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  '• ',
                                  style: TextStyle(
                                    color: Colors.white60,
                                    fontSize: 12,
                                  ),
                                ),
                                Expanded(
                                  child: Text(
                                    source,
                                    style: TextStyle(
                                      color: Colors.white60,
                                      fontSize: 12,
                                      height: 1.3,
                                    ),
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),

                // Muestra la hora del mensaje
                Padding(
                  padding: const EdgeInsets.only(right: 8, bottom: 4),
                  child: Align(
                    alignment: Alignment.bottomRight,
                    child: Text(
                      timeStr,
                      style: TextStyle(
                        color: const Color.fromRGBO(255, 255, 255, 0.7),
                        fontSize: 10,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
