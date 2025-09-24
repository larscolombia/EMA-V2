import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';

class PendingPdfPreview extends StatelessWidget {
  final PdfAttachment pdf;
  final VoidCallback onRemove;
  final bool isUploading;

  const PendingPdfPreview({
    super.key,
    required this.pdf,
    required this.onRemove,
    this.isUploading = false,
  });

  String _formatFileSize(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }

  @override
  Widget build(BuildContext context) {
    final chatController = Get.find<ChatController>();

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 8.0),
      margin: const EdgeInsets.only(left: 16.0, right: 16.0, top: 8.0),
      decoration: BoxDecoration(
        color: Colors.grey[100],
        borderRadius: BorderRadius.circular(8.0),
        border: Border.all(color: Colors.grey[300]!),
      ),
      child: Column(
        children: [
          Row(
            children: [
              // PDF Icon
              Icon(
                Icons.picture_as_pdf,
                color: AppStyles.primaryColor,
                size: 32,
              ),
              const SizedBox(width: 12),

              // PDF Info
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      pdf.fileName,
                      style: TextStyle(fontWeight: FontWeight.bold),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      _formatFileSize(pdf.fileSize),
                      style: TextStyle(color: Colors.grey[600], fontSize: 12),
                    ),
                  ],
                ),
              ),

              // Upload indicator or remove button
              if (isUploading)
                const SizedBox(
                  width: 24,
                  height: 24,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              else
                IconButton(
                  icon: const Icon(Icons.close, color: Colors.grey),
                  onPressed: onRemove,
                  tooltip: 'Eliminar PDF',
                ),
            ],
          ),

          // Focus toggle
          const SizedBox(height: 8),
          Obx(
            () => Row(
              children: [
                Checkbox(
                  value: chatController.focusOnPdfMode.value,
                  onChanged: (_) => chatController.toggleFocusOnPdf(),
                  materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: GestureDetector(
                    onTap: () => chatController.toggleFocusOnPdf(),
                    child: Text(
                      'Preguntar solo sobre este PDF',
                      style: TextStyle(
                        fontSize: 14,
                        color: Colors.grey[700],
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
