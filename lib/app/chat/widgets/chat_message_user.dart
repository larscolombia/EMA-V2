import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../widgets/chat_message_pdf.dart';
import '../widgets/chat_message_image.dart';
import 'animations/slide_in_left.dart';

class ChatMessageUser extends StatelessWidget {
  final ChatMessageModel message;

  const ChatMessageUser({super.key, required this.message});

  @override
  Widget build(BuildContext context) {
    final timeStr = DateFormat('HH:mm').format(message.createdAt);

    return Align(
      alignment: Alignment.centerRight,
      child: SlideInLeft(
        child: Container(
          margin: const EdgeInsets.only(top: 8, bottom: 8, left: 24, right: 12),
          constraints: BoxConstraints(
            maxWidth: MediaQuery.of(context).size.width * 0.75,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            mainAxisSize: MainAxisSize.min,
            children: [
              if (message.text.isNotEmpty)
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: AppStyles.whiteColor,
                    border: Border.all(color: AppStyles.primary900, width: 1),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(message.text, style: AppStyles.chatMessageUser),
                ),
              if (message.imageAttach != null)
                Padding(
                  padding: const EdgeInsets.only(top: 8.0),
                  child: ChatMessageImage(attachment: message.imageAttach!),
                ),
              if (message.attach != null)
                Padding(
                  padding: const EdgeInsets.only(top: 8.0),
                  child: ChatMessagePdf(attachment: message.attach!),
                ),
              Padding(
                padding: const EdgeInsets.only(right: 8, bottom: 4),
                child: Text(
                  timeStr,
                  style: TextStyle(
                    color: const Color.fromRGBO(58, 12, 140, 0.7),
                    fontSize: 10,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
