import 'dart:io';
import 'dart:typed_data';
import 'package:ema_educacion_medica_avanzada/admin/features/books/controllers/vector_stores_controller.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/books/models/vector_store.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/books/models/vector_store_file.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/layout/admin_layout.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/cards/metric_card.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:intl/intl.dart';

class BooksPage extends StatelessWidget {
  const BooksPage({super.key});

  @override
  Widget build(BuildContext context) {
    final controller = Get.put(VectorStoresController());

    return AdminLayout(
      title: 'Base de Conocimiento IA',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Banner informativo + selector de vector store
          Container(
            margin: const EdgeInsets.all(24),
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [
                  AdminColors.primary.withValues(alpha: 0.1),
                  Colors.blue.withValues(alpha: 0.05),
                ],
              ),
              borderRadius: BorderRadius.circular(16),
              border: Border.all(
                color: AdminColors.primary.withValues(alpha: 0.2),
              ),
            ),
            child: Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: const Icon(
                    Icons.psychology,
                    color: AdminColors.primary,
                    size: 32,
                  ),
                ),
                const SizedBox(width: 20),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text(
                        'Documentos para Entrenar la IA',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'Organiza documentos en diferentes vector stores según la especialidad médica. Cada vector store puede ser usado por diferentes asistentes de IA.',
                        style: TextStyle(color: Colors.grey[700], fontSize: 14),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 20),
                Obx(() {
                  if (controller.isLoading.value) {
                    return const CircularProgressIndicator();
                  }
                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 16),
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(12),
                          border: Border.all(color: Colors.grey[300]!),
                        ),
                        child: DropdownButton<int>(
                          value: controller.selectedVectorStore.value?.id,
                          underline: const SizedBox(),
                          hint: const Text('Seleccionar Vector Store'),
                          items: controller.vectorStores.map((vs) {
                            return DropdownMenuItem(
                              value: vs.id,
                              child: Row(
                                children: [
                                  if (vs.isDefault)
                                    const Icon(Icons.star, size: 16, color: Colors.amber),
                                  if (vs.isDefault) const SizedBox(width: 8),
                                  Text(vs.name),
                                ],
                              ),
                            );
                          }).toList(),
                          onChanged: (vsId) {
                            if (vsId != null) {
                              final vs = controller.vectorStores.firstWhere((v) => v.id == vsId);
                              controller.selectVectorStore(vs);
                            }
                          },
                        ),
                      ),
                      const SizedBox(height: 8),
                      Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          TextButton.icon(
                            onPressed: () => _showCreateVectorStoreDialog(context, controller),
                            icon: const Icon(Icons.add, size: 16),
                            label: const Text('Nuevo Vector Store'),
                          ),
                          if (controller.selectedVectorStore.value != null)
                            TextButton.icon(
                              onPressed: () => _showEditVectorStoreDialog(
                                context,
                                controller,
                                controller.selectedVectorStore.value!,
                              ),
                              icon: const Icon(Icons.edit, size: 16),
                              label: const Text('Editar'),
                            ),
                        ],
                      ),
                    ],
                  );
                }),
              ],
            ),
          ),

          // Métricas
          Obx(() {
            final vs = controller.selectedVectorStore.value;
            final files = controller.files;
            final completedFiles = files.where((f) => f.status == 'completed').length;
            final processingFiles = files.where((f) => f.status == 'processing').length;

            return Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Row(
                children: [
                  Expanded(
                    child: MetricCard(
                      title: 'Total Documentos',
                      value: '${vs?.fileCount ?? 0}',
                      icon: Icons.description,
                      color: AdminColors.primary,
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: MetricCard(
                      title: 'Procesados',
                      value: '$completedFiles',
                      icon: Icons.check_circle,
                      color: Colors.green,
                      subtitle: vs != null && vs.fileCount > 0
                          ? '${((completedFiles / vs.fileCount) * 100).toStringAsFixed(1)}% del total'
                          : '0%',
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: MetricCard(
                      title: 'Procesando',
                      value: '$processingFiles',
                      icon: Icons.hourglass_empty,
                      color: Colors.orange,
                      subtitle: processingFiles > 0 ? 'En progreso...' : 'Ninguno',
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: MetricCard(
                      title: 'Tamaño Total',
                      value: vs?.formattedSize ?? '0 B',
                      icon: Icons.storage,
                      color: Colors.blue,
                    ),
                  ),
                ],
              ),
            );
          }),

          const SizedBox(height: 24),

          // Barra de herramientas
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24),
            child: Row(
              children: [
                Obx(() {
                  final vs = controller.selectedVectorStore.value;
                  return Expanded(
                    child: Container(
                      padding: const EdgeInsets.all(16),
                      decoration: BoxDecoration(
                        color: Colors.grey[100],
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Row(
                        children: [
                          Icon(Icons.info_outline, color: Colors.grey[600]),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Text(
                              vs != null
                                  ? '${vs.description} (${vs.category})'
                                  : 'Selecciona un vector store para ver sus archivos',
                              style: TextStyle(color: Colors.grey[700]),
                            ),
                          ),
                        ],
                      ),
                    ),
                  );
                }),
                const SizedBox(width: 16),
                Obx(() {
                  return ElevatedButton.icon(
                    onPressed: controller.selectedVectorStore.value != null &&
                            !controller.isUploading.value
                        ? () => _pickAndUploadFile(controller)
                        : null,
                    icon: controller.isUploading.value
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.cloud_upload),
                    label: Text(
                      controller.isUploading.value ? 'Subiendo...' : 'Subir Documento',
                    ),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: AdminColors.primary,
                      foregroundColor: Colors.white,
                      padding: const EdgeInsets.symmetric(
                        horizontal: 24,
                        vertical: 16,
                      ),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                    ),
                  );
                }),
                const SizedBox(width: 16),
                IconButton(
                  onPressed: () => controller.refreshFiles(),
                  icon: const Icon(Icons.refresh),
                  tooltip: 'Refrescar archivos',
                  color: AdminColors.primary,
                ),
              ],
            ),
          ),

          const SizedBox(height: 24),

          // Tabla de documentos
          Expanded(
            child: Obx(() {
              if (controller.isLoadingFiles.value) {
                return const Center(child: CircularProgressIndicator());
              }

              if (controller.selectedVectorStore.value == null) {
                return Center(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(Icons.folder_open, size: 64, color: Colors.grey[400]),
                      const SizedBox(height: 16),
                      Text(
                        'Selecciona un vector store para ver sus archivos',
                        style: TextStyle(color: Colors.grey[600], fontSize: 16),
                      ),
                    ],
                  ),
                );
              }

              if (controller.files.isEmpty) {
                return Center(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(Icons.description, size: 64, color: Colors.grey[400]),
                      const SizedBox(height: 16),
                      Text(
                        'No hay archivos en este vector store',
                        style: TextStyle(color: Colors.grey[600], fontSize: 16),
                      ),
                      const SizedBox(height: 8),
                      TextButton.icon(
                        onPressed: () => _pickAndUploadFile(controller),
                        icon: const Icon(Icons.cloud_upload),
                        label: const Text('Subir primer archivo'),
                      ),
                    ],
                  ),
                );
              }

              return SingleChildScrollView(
                padding: const EdgeInsets.all(24),
                child: Container(
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(16),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.black.withValues(alpha: 0.05),
                        blurRadius: 10,
                        offset: const Offset(0, 2),
                      ),
                    ],
                  ),
                  child: SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: DataTable(
                      headingRowColor: WidgetStateProperty.all(
                        AdminColors.primary.withValues(alpha: 0.1),
                      ),
                      columnSpacing: 24,
                      horizontalMargin: 24,
                      columns: const [
                      DataColumn(
                        label: Text(
                          'Documento',
                          style: TextStyle(fontWeight: FontWeight.bold),
                        ),
                      ),
                      DataColumn(
                        label: Text(
                          'Tamaño',
                          style: TextStyle(fontWeight: FontWeight.bold),
                        ),
                      ),
                      DataColumn(
                        label: Text(
                          'Estado',
                          style: TextStyle(fontWeight: FontWeight.bold),
                        ),
                      ),
                      DataColumn(
                        label: Text(
                          'Subido por',
                          style: TextStyle(fontWeight: FontWeight.bold),
                        ),
                      ),
                      DataColumn(
                        label: Text(
                          'Fecha',
                          style: TextStyle(fontWeight: FontWeight.bold),
                        ),
                      ),
                      DataColumn(
                        label: Text(
                          'Progreso',
                          style: TextStyle(fontWeight: FontWeight.bold),
                        ),
                      ),
                      DataColumn(
                        label: Text(
                          'Acciones',
                          style: TextStyle(fontWeight: FontWeight.bold),
                        ),
                      ),
                    ],
                    rows: controller.files
                        .map((file) => _buildFileRow(file, controller))
                        .toList(),
                    ),
                  ),
                ),
              );
            }),
          ),
        ],
      ),
    );
  }

  DataRow _buildFileRow(VectorStoreFile file, VectorStoresController controller) {
    Color statusColor = Colors.grey;
    String statusText = file.status;
    
    switch (file.status) {
      case 'completed':
        statusColor = Colors.green;
        statusText = 'Procesado';
        break;
      case 'processing':
        statusColor = Colors.orange;
        statusText = 'Procesando';
        break;
      case 'failed':
        statusColor = Colors.red;
        statusText = 'Error';
        break;
    }

    final dateFormat = DateFormat('dd/MM/yyyy HH:mm');

    return DataRow(
      cells: [
        DataCell(
          Row(
            children: [
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: AdminColors.primary.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(
                  _getFileIcon(file.filename),
                  color: AdminColors.primary,
                  size: 20,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  file.filename,
                  style: const TextStyle(
                    fontWeight: FontWeight.w600,
                    fontSize: 13,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
        ),
        DataCell(
          Text(
            file.formattedSize,
            style: TextStyle(color: Colors.grey[600], fontSize: 13),
          ),
        ),
        DataCell(
          Row(
            children: [
              Container(
                width: 8,
                height: 8,
                decoration: BoxDecoration(
                  color: statusColor,
                  shape: BoxShape.circle,
                ),
              ),
              const SizedBox(width: 8),
              Text(
                statusText,
                style: TextStyle(
                  color: statusColor,
                  fontWeight: FontWeight.w600,
                  fontSize: 13,
                ),
              ),
            ],
          ),
        ),
        DataCell(
          Text(
            file.uploadedBy != null ? 'Admin #${file.uploadedBy}' : 'Sistema',
            style: TextStyle(color: Colors.grey[600], fontSize: 13),
          ),
        ),
        DataCell(
          Text(
            dateFormat.format(file.createdAt),
            style: TextStyle(color: Colors.grey[600], fontSize: 13),
          ),
        ),
        DataCell(
          SizedBox(
            width: 100,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '${(file.uploadProgress * 100).toInt()}%',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: statusColor,
                  ),
                ),
                const SizedBox(height: 4),
                LinearProgressIndicator(
                  value: file.uploadProgress,
                  backgroundColor: Colors.grey[200],
                  valueColor: AlwaysStoppedAnimation<Color>(statusColor),
                  minHeight: 4,
                  borderRadius: BorderRadius.circular(2),
                ),
              ],
            ),
          ),
        ),
        DataCell(
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              IconButton(
                icon: const Icon(Icons.delete, size: 20),
                onPressed: () => _confirmDeleteFile(file, controller),
                tooltip: 'Eliminar',
                color: Colors.red[400],
              ),
            ],
          ),
        ),
      ],
    );
  }

  IconData _getFileIcon(String filename) {
    if (filename.endsWith('.pdf')) return Icons.picture_as_pdf;
    if (filename.endsWith('.txt')) return Icons.description;
    if (filename.endsWith('.md')) return Icons.article;
    return Icons.insert_drive_file;
  }

  Future<void> _pickAndUploadFile(VectorStoresController controller) async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['pdf', 'txt', 'md'],
      withData: true, // Para web: cargar bytes en memoria
    );

    if (result != null) {
      final pickedFile = result.files.single;
      
      // En web usamos bytes, en desktop/mobile usamos path
      if (pickedFile.bytes != null) {
        // Web: crear File temporal desde bytes
        await controller.uploadFileFromBytes(
          pickedFile.bytes!,
          pickedFile.name,
        );
      } else if (pickedFile.path != null) {
        // Desktop/Mobile: usar path directamente
        final file = File(pickedFile.path!);
        await controller.uploadFile(file);
      }
    }
  }

  void _confirmDeleteFile(VectorStoreFile file, VectorStoresController controller) {
    Get.dialog(
      AlertDialog(
        title: const Text('Confirmar eliminación'),
        content: Text('¿Estás seguro de que deseas eliminar "${file.filename}"?'),
        actions: [
          TextButton(
            onPressed: () => Get.back(),
            child: const Text('Cancelar'),
          ),
          ElevatedButton(
            onPressed: () {
              Get.back();
              controller.deleteFile(file.fileId);
            },
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
            child: const Text('Eliminar'),
          ),
        ],
      ),
    );
  }

  void _showCreateVectorStoreDialog(BuildContext context, VectorStoresController controller) {
    final nameController = TextEditingController();
    final descriptionController = TextEditingController();
    String category = 'Medicina General';

    Get.dialog(
      AlertDialog(
        title: const Text('Crear Nuevo Vector Store'),
        content: SizedBox(
          width: 400,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: nameController,
                decoration: const InputDecoration(
                  labelText: 'Nombre',
                  hintText: 'Ej: Cardiología Avanzada',
                ),
              ),
              const SizedBox(height: 16),
              TextField(
                controller: descriptionController,
                decoration: const InputDecoration(
                  labelText: 'Descripción',
                  hintText: 'Ej: Documentos de cardiología clínica',
                ),
                maxLines: 2,
              ),
              const SizedBox(height: 16),
              DropdownButtonFormField<String>(
                value: category,
                decoration: const InputDecoration(labelText: 'Categoría'),
                items: [
                  'Medicina General',
                  'Cardiología',
                  'Neurología',
                  'Pediatría',
                  'Traumatología',
                  'Farmacología',
                  'Anatomía',
                  'Otro',
                ].map((cat) => DropdownMenuItem(value: cat, child: Text(cat))).toList(),
                onChanged: (val) {
                  if (val != null) category = val;
                },
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Get.back(),
            child: const Text('Cancelar'),
          ),
          ElevatedButton(
            onPressed: () {
              if (nameController.text.isNotEmpty && descriptionController.text.isNotEmpty) {
                Get.back();
                controller.createVectorStore(
                  name: nameController.text,
                  description: descriptionController.text,
                  category: category,
                );
              }
            },
            child: const Text('Crear'),
          ),
        ],
      ),
    );
  }

  void _showEditVectorStoreDialog(
    BuildContext context,
    VectorStoresController controller,
    VectorStore vectorStore,
  ) {
    final nameController = TextEditingController(text: vectorStore.name);
    final descriptionController = TextEditingController(text: vectorStore.description);
    String category = vectorStore.category;
    bool isDefault = vectorStore.isDefault;

    Get.dialog(
      StatefulBuilder(
        builder: (context, setState) {
          return AlertDialog(
            title: const Text('Editar Vector Store'),
            content: SizedBox(
              width: 400,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(
                    controller: nameController,
                    decoration: const InputDecoration(labelText: 'Nombre'),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: descriptionController,
                    decoration: const InputDecoration(labelText: 'Descripción'),
                    maxLines: 2,
                  ),
                  const SizedBox(height: 16),
                  DropdownButtonFormField<String>(
                    value: category,
                    decoration: const InputDecoration(labelText: 'Categoría'),
                    items: [
                      'Medicina General',
                      'Cardiología',
                      'Neurología',
                      'Pediatría',
                      'Traumatología',
                      'Farmacología',
                      'Anatomía',
                      'Otro',
                    ].map((cat) => DropdownMenuItem(value: cat, child: Text(cat))).toList(),
                    onChanged: (val) {
                      if (val != null) {
                        setState(() => category = val);
                      }
                    },
                  ),
                  const SizedBox(height: 16),
                  CheckboxListTile(
                    title: const Text('Marcar como predeterminado'),
                    value: isDefault,
                    onChanged: (val) {
                      setState(() => isDefault = val ?? false);
                    },
                  ),
                  if (!vectorStore.isDefault)
                    const SizedBox(height: 16),
                  if (!vectorStore.isDefault)
                    ElevatedButton.icon(
                      onPressed: () {
                        Get.back();
                        Get.dialog(
                          AlertDialog(
                            title: const Text('Confirmar eliminación'),
                            content: Text(
                              '¿Estás seguro de que deseas eliminar "${vectorStore.name}"? '
                              'Todos los archivos asociados también se eliminarán.',
                            ),
                            actions: [
                              TextButton(
                                onPressed: () => Get.back(),
                                child: const Text('Cancelar'),
                              ),
                              ElevatedButton(
                                onPressed: () {
                                  Get.back();
                                  controller.deleteVectorStore(vectorStore.id);
                                },
                                style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
                                child: const Text('Eliminar'),
                              ),
                            ],
                          ),
                        );
                      },
                      icon: const Icon(Icons.delete),
                      label: const Text('Eliminar Vector Store'),
                      style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
                    ),
                ],
              ),
            ),
            actions: [
              TextButton(
                onPressed: () => Get.back(),
                child: const Text('Cancelar'),
              ),
              ElevatedButton(
                onPressed: () {
                  if (nameController.text.isNotEmpty &&
                      descriptionController.text.isNotEmpty) {
                    Get.back();
                    controller.updateVectorStore(
                      vectorStore.id,
                      name: nameController.text,
                      description: descriptionController.text,
                      category: category,
                      isDefault: isDefault != vectorStore.isDefault ? isDefault : null,
                    );
                  }
                },
                child: const Text('Guardar'),
              ),
            ],
          );
        },
      ),
    );
  }
}
