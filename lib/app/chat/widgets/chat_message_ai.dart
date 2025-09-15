import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:gpt_markdown/gpt_markdown.dart';
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

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final String timeStr = DateFormat('HH:mm').format(widget.message.createdAt);
    
    // Debug: imprimir la longitud del texto del mensaje
    print('🔍 [ChatMessageAi] Mensaje recibido - Longitud: ${widget.message.text.length}');
    print('🔍 [ChatMessageAi] Primeros 200 caracteres: ${widget.message.text.length > 200 ? widget.message.text.substring(0, 200) + "..." : widget.message.text}');

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
              left: 8,  // Reducido el margen izquierdo
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
                Padding(
                  padding: const EdgeInsets.all(12),
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(
                      minHeight: 50, // Asegurar altura mínima
                    ),
                    child: ChatMarkdownWrapper(
                      text: widget.message.text,
                      style: const TextStyle(fontSize: 15, color: Colors.white),
                    ),
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
