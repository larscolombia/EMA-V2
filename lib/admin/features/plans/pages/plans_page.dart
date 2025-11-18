import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/layout/admin_layout.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/cards/metric_card.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class PlansPage extends StatelessWidget {
  const PlansPage({super.key});

  @override
  Widget build(BuildContext context) {
    return AdminLayout(
      title: 'Gestión de Planes',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Cards de métricas
          Padding(
            padding: const EdgeInsets.all(24),
            child: Row(
              children: [
                Expanded(
                  child: MetricCard(
                    title: 'Total Planes',
                    value: '3',
                    icon: Icons.credit_card,
                    color: AdminColors.primary,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Usuarios Activos',
                    value: '127',
                    icon: Icons.people,
                    color: Colors.green,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Ingresos Mes',
                    value: '\$3,245',
                    icon: Icons.attach_money,
                    color: Colors.orange,
                  ),
                ),
              ],
            ),
          ),

          // Header con búsqueda y botón crear
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24),
            child: Row(
              children: [
                Expanded(
                  flex: 2,
                  child: TextField(
                    decoration: InputDecoration(
                      hintText: 'Buscar planes...',
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
                ElevatedButton.icon(
                  onPressed: () => Get.snackbar('Info', 'Crear nuevo plan'),
                  icon: const Icon(Icons.add),
                  label: const Text('Nuevo Plan'),
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

          // Tabla de planes
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
                  columns: const [
                    DataColumn(
                      label: Text(
                        'ID',
                        style: TextStyle(fontWeight: FontWeight.bold),
                      ),
                    ),
                    DataColumn(
                      label: Text(
                        'Plan',
                        style: TextStyle(fontWeight: FontWeight.bold),
                      ),
                    ),
                    DataColumn(
                      label: Text(
                        'Precio',
                        style: TextStyle(fontWeight: FontWeight.bold),
                      ),
                    ),
                    DataColumn(
                      label: Text(
                        'Tipo',
                        style: TextStyle(fontWeight: FontWeight.bold),
                      ),
                    ),
                    DataColumn(
                      label: Text(
                        'Consultas',
                        style: TextStyle(fontWeight: FontWeight.bold),
                      ),
                    ),
                    DataColumn(
                      label: Text(
                        'Casos',
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
                        'Acciones',
                        style: TextStyle(fontWeight: FontWeight.bold),
                      ),
                    ),
                  ],
                  rows: [
                    _buildPlanRow(
                      1,
                      'Plan Básico',
                      9.99,
                      'Mensual',
                      10,
                      5,
                      true,
                    ),
                    _buildPlanRow(
                      2,
                      'Plan Premium',
                      29.99,
                      'Mensual',
                      50,
                      25,
                      true,
                    ),
                    _buildPlanRow(
                      3,
                      'Plan Anual',
                      299.99,
                      'Anual',
                      999,
                      999,
                      true,
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

  DataRow _buildPlanRow(
    int id,
    String name,
    double price,
    String type,
    int consultations,
    int cases,
    bool active,
  ) {
    return DataRow(
      cells: [
        DataCell(Text(id.toString())),
        DataCell(
          Row(
            children: [
              Container(
                width: 8,
                height: 8,
                decoration: BoxDecoration(
                  color: active ? Colors.green : Colors.grey,
                  shape: BoxShape.circle,
                ),
              ),
              const SizedBox(width: 12),
              Text(name, style: const TextStyle(fontWeight: FontWeight.w600)),
            ],
          ),
        ),
        DataCell(Text('\$${price.toStringAsFixed(2)}')),
        DataCell(
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: type == 'Mensual' ? Colors.blue[50] : Colors.green[50],
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              type,
              style: TextStyle(
                color: type == 'Mensual' ? Colors.blue[700] : Colors.green[700],
                fontSize: 12,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ),
        DataCell(Text(consultations.toString())),
        DataCell(Text(cases.toString())),
        DataCell(
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: active ? Colors.green[50] : Colors.grey[100],
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              active ? 'Activo' : 'Inactivo',
              style: TextStyle(
                color: active ? Colors.green[700] : Colors.grey[700],
                fontSize: 12,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ),
        DataCell(
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              IconButton(
                icon: const Icon(Icons.edit, size: 20),
                onPressed: () => Get.snackbar('Editar', 'Editar plan $name'),
                tooltip: 'Editar',
                color: AdminColors.primary,
              ),
              IconButton(
                icon: const Icon(Icons.visibility, size: 20),
                onPressed: () => Get.snackbar('Ver', 'Ver detalles de $name'),
                tooltip: 'Ver detalles',
                color: Colors.blue,
              ),
              IconButton(
                icon: const Icon(Icons.delete, size: 20),
                onPressed: () => _confirmDelete(name),
                tooltip: 'Eliminar',
                color: Colors.red[400],
              ),
            ],
          ),
        ),
      ],
    );
  }

  void _confirmDelete(String name) {
    Get.dialog(
      AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        title: Row(
          children: [
            Icon(
              Icons.warning_amber_rounded,
              color: Colors.orange[700],
              size: 28,
            ),
            const SizedBox(width: 12),
            const Text('Confirmar eliminación'),
          ],
        ),
        content: Text(
          '¿Estás seguro de que deseas eliminar el plan "$name"? Esta acción no se puede deshacer.',
        ),
        actions: [
          TextButton(
            onPressed: () => Get.back(),
            child: const Text('Cancelar'),
          ),
          ElevatedButton(
            onPressed: () {
              Get.back();
              Get.snackbar(
                'Eliminado',
                'Plan "$name" eliminado correctamente',
                backgroundColor: Colors.red[100],
                colorText: Colors.red[900],
              );
            },
            style: ElevatedButton.styleFrom(
              backgroundColor: Colors.red,
              foregroundColor: Colors.white,
            ),
            child: const Text('Eliminar'),
          ),
        ],
      ),
    );
  }
}
