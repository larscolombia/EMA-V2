import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../config/config.dart';
import '../../../core/attachments/pdf_attachment.dart';
import '../../profiles/profiles.dart';

class PdfUploaderWidget extends StatelessWidget {
  final Function(PdfAttachment)? onPdfSelected;

  const PdfUploaderWidget({
    super.key,
    this.onPdfSelected,
  });

  void _handlePdfUpload(BuildContext context) async {
    final profileController = Get.find<ProfileController>();

    if (!profileController.canUploadMoreFiles()) {
      Get.snackbar(
        'Límite alcanzado',
        'Has alcanzado el límite de archivos PDF en tu plan actual. Actualiza tu plan para subir más archivos.',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.orange,
        colorText: Colors.white,
        duration: const Duration(seconds: 5),
        mainButton: TextButton(
          onPressed: () => Get.toNamed(Routes.subscriptions.name),
          child: const Text(
            'Actualizar Plan',
            style: TextStyle(color: Colors.white),
          ),
        ),
      );
      return;
    }

    FilePickerResult? result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['pdf'],
      withData: true,
    );

    if (result != null) {
      final file = result.files.first;
      if (file.path != null) {
        final pdfAttachment = PdfAttachment(
          uid: DateTime.now().toString(),
          fileName: file.name,
          mimeType: 'application/pdf',
          filePath: file.path!,
          fileSize: file.size,
        );

        // Notificar la selección del archivo
        onPdfSelected?.call(pdfAttachment);

        // Mostrar mensaje de éxito
        Get.snackbar(
          'Éxito',
          'PDF cargado exitosamente',
          snackPosition: SnackPosition.TOP,
          backgroundColor: Colors.green,
          colorText: Colors.white,
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 20.0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _buildUploadArea(context),

          // const SizedBox(height: 24.0),

          // _buildMyPdfsButton(),

          const SizedBox(height: 12.0),

          // Divider(indent: 12, endIndent: 12),

          // const SizedBox(height: 12.0),

          //_buildPremiumButton(),
        ],
      ),
    );
  }

  Widget _buildUploadArea(BuildContext context) {
    return Obx(() {
      final profileController = Get.find<ProfileController>();
      final canUpload = profileController.canUploadMoreFiles();

      return Container(
        decoration: BoxDecoration(
          color: canUpload ? AppStyles.grey220 : Colors.grey[300],
          border: Border.all(color: AppStyles.grey150),
          borderRadius: BorderRadius.circular(16.0),
          boxShadow: [
            BoxShadow(
              color: const Color.fromRGBO(158, 158, 158, 0.1),
              blurRadius: 8,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        child: Material(
          color: Colors.transparent,
          child: InkWell(
            onTap: canUpload
                ? () => _handlePdfUpload(context)
                : () {
                    Get.snackbar(
                      'Límite alcanzado',
                      'Has alcanzado el límite de archivos PDF en tu plan actual. Actualiza tu plan para subir más archivos.',
                      snackPosition: SnackPosition.TOP,
                      backgroundColor: Colors.orange,
                      colorText: Colors.white,
                      duration: const Duration(seconds: 5),
                      mainButton: TextButton(
                        onPressed: () => Get.toNamed(Routes.subscriptions.name),
                        child: const Text(
                          'Actualizar Plan',
                          style: TextStyle(color: Colors.white),
                        ),
                      ),
                    );
                  },
            borderRadius: BorderRadius.circular(20.0),
            child: Padding(
              padding:
                  const EdgeInsets.symmetric(vertical: 30.0, horizontal: 20.0),
              child: Column(
                children: [
                  Icon(
                    Icons.upload_file_rounded,
                    size: 50.0,
                    color: canUpload ? AppStyles.grey200 : Colors.grey[400],
                  ),
                  Text(
                    'Subir PDF',
                    style: TextStyle(
                      fontSize: 16.0,
                      fontWeight: FontWeight.w500,
                      color: canUpload ? Colors.grey[800] : Colors.grey[600],
                    ),
                  ),
                  Text(
                    canUpload
                        ? 'Toca para seleccionar un archivo'
                        : 'Límite de archivos alcanzado',
                    style: TextStyle(
                      fontSize: 14.0,
                      color: Colors.grey[600],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      );
    });
  }

  // Widget _buildMyPdfsButton() {
  //   return ElevatedButton(
  //     style: ElevatedButton.styleFrom(
  //       foregroundColor: AppStyles.primaryColor,
  //       backgroundColor: AppStyles.whiteColor,
  //       elevation: 2,
  //       padding: const EdgeInsets.symmetric(vertical: 16.0),
  //       side: BorderSide(color: AppStyles.primaryColor),
  //       shape: RoundedRectangleBorder(
  //         borderRadius: BorderRadius.circular(30.0),
  //       ),
  //     ),
  //     onPressed: () {},
  //     child: const Text(
  //       'Mis PDF',
  //       style: TextStyle(
  //         fontSize: 16.0,
  //         fontWeight: FontWeight.w600,
  //       ),
  //     ),
  //   );
  // }

  /*Widget _buildPremiumButton() {
    return ElevatedButton(
      style: ElevatedButton.styleFrom(
        foregroundColor: AppStyles.primary900,
        backgroundColor: AppStyles.whiteColor,
        elevation: 2,
        padding: const EdgeInsets.symmetric(vertical: 16.0),
        side: BorderSide(color: AppStyles.primary900, width: 1.5),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(30.0),
        ),
      ),
      onPressed: () {
        Get.toNamed(Routes.subscriptions.name);
      },
      child: const Text(
        'Obtener Premium',
        style: TextStyle(
          fontSize: 16.0,
          fontWeight: FontWeight.bold,
          color: AppStyles.primary900,
        ),
      ),
    );
  }*/
}
