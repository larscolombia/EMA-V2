import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/layout/admin_layout.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/cards/metric_card.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class BooksPage extends StatelessWidget {
  const BooksPage({super.key});

  @override
  Widget build(BuildContext context) {
    return AdminLayout(
      title: 'Base de Conocimiento IA',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Banner informativo
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
                        'Los documentos subidos aquí alimentan la base de conocimiento de la IA del backend. Se utilizan para mejorar las respuestas en casos clínicos, consultas y cuestionarios.',
                        style: TextStyle(color: Colors.grey[700], fontSize: 14),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),

          // Métricas
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24),
            child: Row(
              children: [
                Expanded(
                  child: MetricCard(
                    title: 'Total Documentos',
                    value: '247',
                    icon: Icons.description,
                    color: AdminColors.primary,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Procesados',
                    value: '189',
                    icon: Icons.check_circle,
                    color: Colors.green,
                    subtitle: '76.5% del total',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'En Cola',
                    value: '58',
                    icon: Icons.hourglass_empty,
                    color: Colors.orange,
                    subtitle: 'Procesando...',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Tamaño Total',
                    value: '4.8 GB',
                    icon: Icons.storage,
                    color: Colors.blue,
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 24),

          // Barra de herramientas
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    decoration: InputDecoration(
                      hintText: 'Buscar documentos...',
                      prefixIcon: const Icon(Icons.search),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                      filled: true,
                      fillColor: Colors.grey[50],
                    ),
                  ),
                ),
                const SizedBox(width: 16),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  decoration: BoxDecoration(
                    border: Border.all(color: Colors.grey[300]!),
                    borderRadius: BorderRadius.circular(12),
                    color: Colors.white,
                  ),
                  child: DropdownButton<String>(
                    value: 'Todos',
                    underline: const SizedBox(),
                    items:
                        ['Todos', 'Procesados', 'En Cola', 'Con Errores']
                            .map(
                              (e) => DropdownMenuItem(value: e, child: Text(e)),
                            )
                            .toList(),
                    onChanged: (v) => Get.snackbar('Filtro', 'Estado: $v'),
                  ),
                ),
                const SizedBox(width: 16),
                ElevatedButton.icon(
                  onPressed:
                      () => Get.snackbar(
                        'Subir',
                        'Función de backend - Próximamente',
                      ),
                  icon: const Icon(Icons.cloud_upload),
                  label: const Text('Subir Documentos'),
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
                ),
              ],
            ),
          ),

          const SizedBox(height: 24),

          // Tabla de documentos
          Expanded(
            child: SingleChildScrollView(
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
                        'Categoría',
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
                        'Subido',
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
                  rows: [
                    _buildDocRow(
                      'Harrison Principios Medicina Interna.pdf',
                      'Medicina Interna',
                      '48.2 MB',
                      'Procesado',
                      '10/11/2024',
                      1.0,
                      Colors.green,
                    ),
                    _buildDocRow(
                      'Atlas Anatomía Netter 7ma Edición.pdf',
                      'Anatomía',
                      '125.8 MB',
                      'Procesado',
                      '09/11/2024',
                      1.0,
                      Colors.green,
                    ),
                    _buildDocRow(
                      'Guía Práctica Cardiología ESC 2024.pdf',
                      'Cardiología',
                      '12.5 MB',
                      'En Cola',
                      '08/11/2024',
                      0.45,
                      Colors.orange,
                    ),
                    _buildDocRow(
                      'Neurología Clínica Fitzgerald.pdf',
                      'Neurología',
                      '34.1 MB',
                      'Procesado',
                      '07/11/2024',
                      1.0,
                      Colors.green,
                    ),
                    _buildDocRow(
                      'Farmacología Básica y Clínica.pdf',
                      'Farmacología',
                      '28.9 MB',
                      'En Cola',
                      '06/11/2024',
                      0.72,
                      Colors.orange,
                    ),
                    _buildDocRow(
                      'Pediatría Nelson Esencial.pdf',
                      'Pediatría',
                      '41.3 MB',
                      'Procesado',
                      '05/11/2024',
                      1.0,
                      Colors.green,
                    ),
                    _buildDocRow(
                      'ECG Interpretación Práctica.pdf',
                      'Cardiología',
                      '8.7 MB',
                      'Error',
                      '04/11/2024',
                      0.0,
                      Colors.red,
                    ),
                    _buildDocRow(
                      'Traumatología y Ortopedia.pdf',
                      'Traumatología',
                      '52.4 MB',
                      'Procesado',
                      '03/11/2024',
                      1.0,
                      Colors.green,
                    ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  DataRow _buildDocRow(
    String name,
    String category,
    String size,
    String status,
    String uploadDate,
    double progress,
    Color statusColor,
  ) {
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
                child: const Icon(
                  Icons.picture_as_pdf,
                  color: AdminColors.primary,
                  size: 20,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  name,
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
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            decoration: BoxDecoration(
              color: Colors.blue.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              category,
              style: const TextStyle(
                color: Colors.blue,
                fontSize: 12,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ),
        DataCell(
          Text(size, style: TextStyle(color: Colors.grey[600], fontSize: 13)),
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
                status,
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
            uploadDate,
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
                  '${(progress * 100).toInt()}%',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: statusColor,
                  ),
                ),
                const SizedBox(height: 4),
                LinearProgressIndicator(
                  value: progress,
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
              if (status == 'Error')
                IconButton(
                  icon: const Icon(Icons.refresh, size: 20),
                  onPressed:
                      () => Get.snackbar('Reprocesar', 'Reprocesando: $name'),
                  tooltip: 'Reprocesar',
                  color: Colors.blue,
                ),
              IconButton(
                icon: const Icon(Icons.download, size: 20),
                onPressed:
                    () => Get.snackbar('Descargar', 'Descargando: $name'),
                tooltip: 'Descargar',
                color: AdminColors.primary,
              ),
              IconButton(
                icon: const Icon(Icons.delete, size: 20),
                onPressed: () => Get.snackbar('Eliminar', 'Eliminando: $name'),
                tooltip: 'Eliminar',
                color: Colors.red[400],
              ),
            ],
          ),
        ),
      ],
    );
  }
}
