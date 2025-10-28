import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../config/config.dart';
import '../../../core/attachments/image_attachment.dart';
import '../../profiles/profiles.dart';

class ImageUploaderWidget extends StatelessWidget {
  final Function(ImageAttachment)? onImageSelected;

  const ImageUploaderWidget({super.key, this.onImageSelected});

  void _handleImageUpload(BuildContext context) async {
    final profileController = Get.find<ProfileController>();

    if (!profileController.canUploadMoreFiles()) {
      Get.snackbar(
        'Límite alcanzado',
        'Has alcanzado el límite de archivos en tu plan actual. Actualiza tu plan para subir más archivos.',
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
      allowedExtensions: ['jpg', 'jpeg', 'png', 'gif', 'webp'],
      withData: true,
    );

    if (result != null) {
      final file = result.files.first;
      if (file.path != null) {
        // Verificar tamaño máximo (20 MB)
        const maxSize = 20 * 1024 * 1024; // 20 MB en bytes
        if (file.size > maxSize) {
          Get.snackbar(
            'Archivo muy grande',
            'La imagen no debe superar los 20 MB',
            snackPosition: SnackPosition.TOP,
            backgroundColor: Colors.red,
            colorText: Colors.white,
          );
          return;
        }

        // Determinar el tipo MIME basado en la extensión
        String mimeType = 'image/jpeg'; // default
        final extension = file.extension?.toLowerCase();
        if (extension == 'png') {
          mimeType = 'image/png';
        } else if (extension == 'gif') {
          mimeType = 'image/gif';
        } else if (extension == 'webp') {
          mimeType = 'image/webp';
        }

        final imageAttachment = ImageAttachment(
          uid: DateTime.now().toString(),
          fileName: file.name,
          mimeType: mimeType,
          filePath: file.path!,
          fileSize: file.size,
        );

        // Notificar la selección del archivo
        onImageSelected?.call(imageAttachment);

        // Mostrar mensaje de éxito
        Get.snackbar(
          'Éxito',
          'Imagen cargada exitosamente',
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
        children: [_buildUploadArea(context), const SizedBox(height: 12.0)],
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
            onTap:
                canUpload
                    ? () => _handleImageUpload(context)
                    : () {
                      Get.snackbar(
                        'Límite alcanzado',
                        'Has alcanzado el límite de archivos en tu plan actual. Actualiza tu plan para subir más archivos.',
                        snackPosition: SnackPosition.TOP,
                        backgroundColor: Colors.orange,
                        colorText: Colors.white,
                        duration: const Duration(seconds: 5),
                        mainButton: TextButton(
                          onPressed:
                              () => Get.toNamed(Routes.subscriptions.name),
                          child: const Text(
                            'Actualizar Plan',
                            style: TextStyle(color: Colors.white),
                          ),
                        ),
                      );
                    },
            borderRadius: BorderRadius.circular(20.0),
            child: Padding(
              padding: const EdgeInsets.symmetric(
                vertical: 30.0,
                horizontal: 20.0,
              ),
              child: Column(
                children: [
                  Icon(
                    Icons.image_rounded,
                    size: 50.0,
                    color: canUpload ? AppStyles.grey200 : Colors.grey[400],
                  ),
                  Text(
                    'Subir Imagen',
                    style: TextStyle(
                      fontSize: 16.0,
                      fontWeight: FontWeight.w500,
                      color: canUpload ? Colors.grey[800] : Colors.grey[600],
                    ),
                  ),
                  Text(
                    canUpload
                        ? 'Toca para seleccionar una imagen (máx. 20MB)'
                        : 'Límite de archivos alcanzado',
                    style: TextStyle(fontSize: 14.0, color: Colors.grey[600]),
                  ),
                ],
              ),
            ),
          ),
        ),
      );
    });
  }
}
