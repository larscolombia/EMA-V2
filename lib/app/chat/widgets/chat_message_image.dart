import 'package:flutter/material.dart';
import 'dart:io';
import '../../../core/attachments/image_attachment.dart';

class ChatMessageImage extends StatelessWidget {
  final ImageAttachment attachment;

  const ChatMessageImage({super.key, required this.attachment});

  void _showFullImage(BuildContext context) {
    final file = File(attachment.filePath);

    if (!file.existsSync()) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('No se puede encontrar la imagen')),
      );
      return;
    }

    Navigator.push(
      context,
      MaterialPageRoute<void>(
        builder:
            (BuildContext context) => Scaffold(
              backgroundColor: Colors.black,
              appBar: AppBar(
                title: Text(attachment.fileName),
                backgroundColor: Colors.black,
              ),
              body: Center(
                child: InteractiveViewer(
                  panEnabled: true,
                  minScale: 0.5,
                  maxScale: 4.0,
                  child: Image.file(
                    file,
                    fit: BoxFit.contain,
                    errorBuilder: (context, error, stackTrace) {
                      return const Center(
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Icon(
                              Icons.error_outline,
                              color: Colors.white,
                              size: 48,
                            ),
                            SizedBox(height: 16),
                            Text(
                              'Error al cargar la imagen',
                              style: TextStyle(color: Colors.white),
                            ),
                          ],
                        ),
                      );
                    },
                  ),
                ),
              ),
            ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final file = File(attachment.filePath);

    return Card(
      elevation: 2,
      margin: const EdgeInsets.symmetric(vertical: 4),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => _showFullImage(context),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Preview de la imagen
            Container(
              constraints: const BoxConstraints(
                maxHeight: 200,
                maxWidth: double.infinity,
              ),
              child: Image.file(
                file,
                fit: BoxFit.cover,
                width: double.infinity,
                errorBuilder: (context, error, stackTrace) {
                  return Container(
                    height: 150,
                    color: Colors.grey[300],
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(
                          Icons.broken_image,
                          size: 48,
                          color: Colors.grey[600],
                        ),
                        const SizedBox(height: 8),
                        Text(
                          'Error al cargar imagen',
                          style: TextStyle(color: Colors.grey[600]),
                        ),
                      ],
                    ),
                  );
                },
              ),
            ),
            // Información de la imagen
            Padding(
              padding: const EdgeInsets.all(12),
              child: Row(
                children: [
                  Icon(Icons.image_rounded, color: Colors.blue[700], size: 20),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          attachment.fileName,
                          style: const TextStyle(
                            fontSize: 13,
                            fontWeight: FontWeight.w500,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        Text(
                          'Toca para ampliar',
                          style: TextStyle(
                            fontSize: 11,
                            color: Colors.grey[600],
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
