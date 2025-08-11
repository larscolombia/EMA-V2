import 'package:flutter/material.dart';
import 'dart:io';
import '../../../core/attachments/pdf_attachment.dart';
import 'package:flutter_pdfview/flutter_pdfview.dart';

class ChatMessagePdf extends StatelessWidget {
  final PdfAttachment attachment;

  const ChatMessagePdf({
    super.key,
    required this.attachment,
  });

  void _showPdfViewer(BuildContext context) {
    final file = File(attachment.filePath);

    if (!file.existsSync()) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('No se puede encontrar el archivo PDF')),
      );
      return;
    }

    Navigator.push(
      context,
      MaterialPageRoute<void>(
        builder: (BuildContext context) => Scaffold(
          appBar: AppBar(
            title: Text(attachment.fileName),
          ),
          body: PDFView(
            filePath: attachment.filePath,
            enableSwipe: true,
            swipeHorizontal: false,
            autoSpacing: true,
            pageSnap: true,
            pageFling: true,
            onError: (error) {
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('Error al abrir el PDF: $error')),
              );
              Navigator.pop(context);
            },
            onPageError: (page, error) {},
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 2,
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: InkWell(
        onTap: () => _showPdfViewer(context),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: Colors.red.shade100,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(Icons.picture_as_pdf,
                    color: Colors.red[700], size: 24),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      attachment.fileName,
                      style: const TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w500,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      'Toca para abrir',
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.grey[600],
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
