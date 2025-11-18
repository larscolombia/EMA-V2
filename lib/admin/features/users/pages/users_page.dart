import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/layout/admin_layout.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/cards/metric_card.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class UsersPage extends StatelessWidget {
  const UsersPage({super.key});

  @override
  Widget build(BuildContext context) {
    return AdminLayout(
      title: 'Gestión de Usuarios',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Métricas
          Padding(
            padding: const EdgeInsets.all(24),
            child: Row(
              children: [
                Expanded(
                  child: MetricCard(
                    title: 'Total Usuarios',
                    value: '1,247',
                    icon: Icons.people,
                    color: AdminColors.primary,
                    subtitle: '+12% este mes',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Usuarios Activos',
                    value: '892',
                    icon: Icons.check_circle,
                    color: Colors.green,
                    subtitle: '71.5% del total',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Nuevos (mes)',
                    value: '156',
                    icon: Icons.person_add,
                    color: Colors.blue,
                    subtitle: '+8% vs mes anterior',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Usuarios Premium',
                    value: '567',
                    icon: Icons.star,
                    color: Colors.orange,
                    subtitle: '45.5% del total',
                  ),
                ),
              ],
            ),
          ),

          // Filtros y búsqueda
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24),
            child: Row(
              children: [
                Expanded(
                  flex: 2,
                  child: TextField(
                    decoration: InputDecoration(
                      hintText: 'Buscar por nombre, email o ID...',
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
                    value: 'Todos los estados',
                    underline: const SizedBox(),
                    items:
                        [
                              'Todos los estados',
                              'Activos',
                              'Inactivos',
                              'Suspendidos',
                            ]
                            .map(
                              (e) => DropdownMenuItem(value: e, child: Text(e)),
                            )
                            .toList(),
                    onChanged: (v) => Get.snackbar('Filtro', 'Estado: $v'),
                  ),
                ),
                const SizedBox(width: 12),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  decoration: BoxDecoration(
                    border: Border.all(color: Colors.grey[300]!),
                    borderRadius: BorderRadius.circular(12),
                    color: Colors.white,
                  ),
                  child: DropdownButton<String>(
                    value: 'Todos los planes',
                    underline: const SizedBox(),
                    items:
                        [
                              'Todos los planes',
                              'Plan Básico',
                              'Plan Premium',
                              'Plan Anual',
                            ]
                            .map(
                              (e) => DropdownMenuItem(value: e, child: Text(e)),
                            )
                            .toList(),
                    onChanged: (v) => Get.snackbar('Filtro', 'Plan: $v'),
                  ),
                ),
                const SizedBox(width: 16),
                ElevatedButton.icon(
                  onPressed:
                      () => Get.snackbar('Exportar', 'Exportando usuarios...'),
                  icon: const Icon(Icons.download, size: 18),
                  label: const Text('Exportar'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.green,
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 20,
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

          // Tabla de usuarios
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
                child: Column(
                  children: [
                    DataTable(
                      headingRowColor: WidgetStateProperty.all(
                        AdminColors.primary.withValues(alpha: 0.1),
                      ),
                      columnSpacing: 24,
                      horizontalMargin: 24,
                      columns: const [
                        DataColumn(
                          label: Text(
                            'ID',
                            style: TextStyle(fontWeight: FontWeight.bold),
                          ),
                        ),
                        DataColumn(
                          label: Text(
                            'Usuario',
                            style: TextStyle(fontWeight: FontWeight.bold),
                          ),
                        ),
                        DataColumn(
                          label: Text(
                            'Email',
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
                            'Estado',
                            style: TextStyle(fontWeight: FontWeight.bold),
                          ),
                        ),
                        DataColumn(
                          label: Text(
                            'Registro',
                            style: TextStyle(fontWeight: FontWeight.bold),
                          ),
                        ),
                        DataColumn(
                          label: Text(
                            'Actividad',
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
                        _buildUserRow(
                          1,
                          'Dr. Juan Pérez',
                          'juan.perez@hospital.com',
                          'Premium',
                          true,
                          '15/10/2024',
                          '2 hrs',
                          'assets/avatar1.png',
                        ),
                        _buildUserRow(
                          2,
                          'Dra. María García',
                          'maria.garcia@clinica.com',
                          'Básico',
                          true,
                          '12/10/2024',
                          '5 hrs',
                          'assets/avatar2.png',
                        ),
                        _buildUserRow(
                          3,
                          'Dr. Carlos López',
                          'carlos.lopez@email.com',
                          'Anual',
                          true,
                          '08/10/2024',
                          '1 día',
                          'assets/avatar3.png',
                        ),
                        _buildUserRow(
                          4,
                          'Dra. Ana Martínez',
                          'ana.martinez@hospital.com',
                          'Premium',
                          false,
                          '05/10/2024',
                          '3 días',
                          'assets/avatar4.png',
                        ),
                        _buildUserRow(
                          5,
                          'Dr. Luis Rodríguez',
                          'luis.rodriguez@email.com',
                          'Básico',
                          true,
                          '01/10/2024',
                          '1 hr',
                          'assets/avatar5.png',
                        ),
                        _buildUserRow(
                          6,
                          'Dra. Carmen Silva',
                          'carmen.silva@clinica.com',
                          'Premium',
                          true,
                          '28/09/2024',
                          '4 hrs',
                          'assets/avatar6.png',
                        ),
                        _buildUserRow(
                          7,
                          'Dr. Pedro Gómez',
                          'pedro.gomez@hospital.com',
                          'Básico',
                          false,
                          '25/09/2024',
                          '1 sem',
                          'assets/avatar7.png',
                        ),
                        _buildUserRow(
                          8,
                          'Dra. Laura Díaz',
                          'laura.diaz@email.com',
                          'Anual',
                          true,
                          '20/09/2024',
                          '2 días',
                          'assets/avatar8.png',
                        ),
                      ],
                    ),
                    const SizedBox(height: 16),
                    // Paginación
                    Padding(
                      padding: const EdgeInsets.all(16),
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Text(
                            'Mostrando 1-8 de 1,247 usuarios',
                            style: TextStyle(
                              color: Colors.grey[600],
                              fontSize: 14,
                            ),
                          ),
                          Row(
                            children: [
                              IconButton(
                                icon: const Icon(Icons.chevron_left),
                                onPressed:
                                    () => Get.snackbar(
                                      'Paginación',
                                      'Página anterior',
                                    ),
                                tooltip: 'Anterior',
                              ),
                              Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 16,
                                  vertical: 8,
                                ),
                                decoration: BoxDecoration(
                                  color: AdminColors.primary.withValues(
                                    alpha: 0.1,
                                  ),
                                  borderRadius: BorderRadius.circular(8),
                                ),
                                child: const Text(
                                  '1',
                                  style: TextStyle(
                                    color: AdminColors.primary,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ),
                              const SizedBox(width: 8),
                              TextButton(
                                onPressed:
                                    () =>
                                        Get.snackbar('Paginación', 'Página 2'),
                                child: const Text('2'),
                              ),
                              const SizedBox(width: 8),
                              TextButton(
                                onPressed:
                                    () =>
                                        Get.snackbar('Paginación', 'Página 3'),
                                child: const Text('3'),
                              ),
                              const SizedBox(width: 8),
                              const Text('...'),
                              const SizedBox(width: 8),
                              TextButton(
                                onPressed:
                                    () => Get.snackbar(
                                      'Paginación',
                                      'Última página',
                                    ),
                                child: const Text('157'),
                              ),
                              IconButton(
                                icon: const Icon(Icons.chevron_right),
                                onPressed:
                                    () => Get.snackbar(
                                      'Paginación',
                                      'Página siguiente',
                                    ),
                                tooltip: 'Siguiente',
                              ),
                            ],
                          ),
                        ],
                      ),
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

  DataRow _buildUserRow(
    int id,
    String name,
    String email,
    String plan,
    bool active,
    String registerDate,
    String lastActivity,
    String avatar,
  ) {
    Color planColor;
    switch (plan) {
      case 'Premium':
        planColor = Colors.purple;
        break;
      case 'Anual':
        planColor = Colors.green;
        break;
      default:
        planColor = Colors.blue;
    }

    return DataRow(
      cells: [
        DataCell(Text('#$id')),
        DataCell(
          Row(
            children: [
              CircleAvatar(
                radius: 16,
                backgroundColor: AdminColors.primary.withValues(alpha: 0.1),
                child: Text(
                  name.split(' ')[1][0],
                  style: const TextStyle(
                    color: AdminColors.primary,
                    fontWeight: FontWeight.bold,
                    fontSize: 14,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    name,
                    style: const TextStyle(
                      fontWeight: FontWeight.w600,
                      fontSize: 14,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
        DataCell(
          Text(email, style: TextStyle(color: Colors.grey[600], fontSize: 13)),
        ),
        DataCell(
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: planColor.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (plan == 'Premium' || plan == 'Anual')
                  Icon(Icons.star, size: 14, color: planColor),
                if (plan == 'Premium' || plan == 'Anual')
                  const SizedBox(width: 4),
                Text(
                  plan,
                  style: TextStyle(
                    color: planColor,
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          ),
        ),
        DataCell(
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 8,
                height: 8,
                decoration: BoxDecoration(
                  color: active ? Colors.green : Colors.grey,
                  shape: BoxShape.circle,
                ),
              ),
              const SizedBox(width: 8),
              Text(
                active ? 'Activo' : 'Inactivo',
                style: TextStyle(
                  color: active ? Colors.green[700] : Colors.grey[600],
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
        DataCell(
          Text(
            registerDate,
            style: TextStyle(color: Colors.grey[600], fontSize: 13),
          ),
        ),
        DataCell(
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            decoration: BoxDecoration(
              color: Colors.grey[100],
              borderRadius: BorderRadius.circular(8),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.access_time, size: 14, color: Colors.grey[600]),
                const SizedBox(width: 4),
                Text(
                  lastActivity,
                  style: TextStyle(color: Colors.grey[700], fontSize: 12),
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
                icon: const Icon(Icons.visibility, size: 20),
                onPressed: () => _showUserDetails(name, email),
                tooltip: 'Ver detalles',
                color: AdminColors.primary,
              ),
              IconButton(
                icon: const Icon(Icons.edit, size: 20),
                onPressed:
                    () => Get.snackbar('Editar', 'Editar usuario: $name'),
                tooltip: 'Editar',
                color: Colors.blue,
              ),
              IconButton(
                icon: Icon(active ? Icons.block : Icons.check_circle, size: 20),
                onPressed:
                    () => Get.snackbar(
                      active ? 'Suspender' : 'Activar',
                      '${active ? 'Suspender' : 'Activar'} usuario: $name',
                    ),
                tooltip: active ? 'Suspender' : 'Activar',
                color: active ? Colors.orange : Colors.green,
              ),
            ],
          ),
        ),
      ],
    );
  }

  void _showUserDetails(String name, String email) {
    Get.dialog(
      AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        title: Row(
          children: [
            CircleAvatar(
              radius: 24,
              backgroundColor: AdminColors.primary.withValues(alpha: 0.1),
              child: Text(
                name.split(' ')[1][0],
                style: const TextStyle(
                  color: AdminColors.primary,
                  fontWeight: FontWeight.bold,
                  fontSize: 20,
                ),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(name, style: const TextStyle(fontSize: 18)),
                  Text(
                    email,
                    style: TextStyle(
                      fontSize: 14,
                      color: Colors.grey[600],
                      fontWeight: FontWeight.normal,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
        content: SizedBox(
          width: 500,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _buildDetailRow(
                'Plan actual',
                'Plan Premium',
                Icons.star,
                Colors.purple,
              ),
              const Divider(),
              _buildDetailRow(
                'Fecha de registro',
                '15 de octubre, 2024',
                Icons.calendar_today,
                Colors.blue,
              ),
              const Divider(),
              _buildDetailRow(
                'Última actividad',
                'Hace 2 horas',
                Icons.access_time,
                Colors.green,
              ),
              const Divider(),
              _buildDetailRow(
                'Casos resueltos',
                '47 casos',
                Icons.medical_services,
                Colors.red,
              ),
              const Divider(),
              _buildDetailRow(
                'Cuestionarios',
                '83 completados',
                Icons.quiz,
                Colors.orange,
              ),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Get.back(), child: const Text('Cerrar')),
          ElevatedButton(
            onPressed: () {
              Get.back();
              Get.snackbar('Editar', 'Editar usuario: $name');
            },
            style: ElevatedButton.styleFrom(
              backgroundColor: AdminColors.primary,
              foregroundColor: Colors.white,
            ),
            child: const Text('Editar Usuario'),
          ),
        ],
      ),
    );
  }

  Widget _buildDetailRow(
    String label,
    String value,
    IconData icon,
    Color color,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Icon(icon, color: color, size: 20),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  label,
                  style: TextStyle(color: Colors.grey[600], fontSize: 12),
                ),
                const SizedBox(height: 2),
                Text(
                  value,
                  style: const TextStyle(
                    fontWeight: FontWeight.w600,
                    fontSize: 14,
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
