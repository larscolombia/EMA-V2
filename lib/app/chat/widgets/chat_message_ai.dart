import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:gpt_markdown/gpt_markdown.dart';
import 'package:url_launcher/url_launcher.dart';
import 'animations/slide_in_left.dart';

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

    return Align(
      alignment: Alignment.centerLeft,
      child: SlideInLeft(
        child: FadeTransition(
          opacity: _fadeAnimation,
          child: Container(
            margin: const EdgeInsets.only(
              top: 8,
              bottom: 8,
              right: 24,
              left: 12,
            ),
            constraints: BoxConstraints(
              maxWidth: MediaQuery.of(context).size.width * 0.75,
            ),
            decoration: BoxDecoration(
              color: const Color.fromRGBO(58, 12, 140, 0.9),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: const EdgeInsets.all(12),
                  child: GptMarkdown(
                    widget.message.text,
                    // Parámetros adicionales para personalizar la renderización:
                    textAlign: TextAlign.justify,
                    textScaler: const TextScaler.linear(1),
                    style: const TextStyle(fontSize: 15, color: Colors.white),

                    // onLinkTab eliminado por incompatibilidad con la versión actual de gpt_markdown
                    // Además, puedes sobreescribir la forma en que se muestra el enlace utilizando linkBuilder
                    linkBuilder: (context, label, path, style) {
                      return Text(
                        label.toString(),
                        style: style.copyWith(
                          color: Colors.blue,
                          decoration: TextDecoration.underline,
                        ),
                      );
                    },
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
