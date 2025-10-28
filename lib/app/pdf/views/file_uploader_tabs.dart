import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../config/config.dart';
import '../../../core/attachments/image_attachment.dart';
import '../../../core/attachments/pdf_attachment.dart';
import '../../chat/controllers/chat_controller.dart';
import 'image_uploader_widget.dart';
import 'pdf_uploader_widget.dart';

class FileUploaderTabs extends StatelessWidget {
  const FileUploaderTabs({super.key});

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 2,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          // Tab Bar
          Container(
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(color: Colors.grey[300]!, width: 1),
              ),
            ),
            child: TabBar(
              labelColor: AppStyles.primaryColor,
              unselectedLabelColor: Colors.grey[600],
              indicatorColor: AppStyles.primaryColor,
              indicatorWeight: 3,
              tabs: const [
                Tab(icon: Icon(Icons.picture_as_pdf), text: 'PDF'),
                Tab(icon: Icon(Icons.image_rounded), text: 'Imagen'),
              ],
            ),
          ),

          // Tab Bar View
          SizedBox(
            height: 250, // Altura fija para el contenido
            child: TabBarView(
              children: [
                // PDF Uploader
                PdfUploaderWidget(
                  onPdfSelected: (PdfAttachment pdfAttachment) {
                    final chatController = Get.find<ChatController>();
                    chatController.attachPdf(pdfAttachment);
                    Get.back(); // Cerrar el overlay después de seleccionar
                  },
                ),

                // Image Uploader
                ImageUploaderWidget(
                  onImageSelected: (ImageAttachment imageAttachment) {
                    final chatController = Get.find<ChatController>();
                    chatController.attachImage(imageAttachment);
                    Get.back(); // Cerrar el overlay después de seleccionar
                  },
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
