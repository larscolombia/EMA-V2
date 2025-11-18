import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/layout/admin_layout.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/cards/metric_card.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class DashboardPage extends StatelessWidget {
  const DashboardPage({super.key});

  @override
  Widget build(BuildContext context) {
    return AdminLayout(
      title: 'Dashboard',
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Métricas principales - Fila 1
            Row(
              children: [
                Expanded(
                  child: MetricCard(
                    title: 'Usuarios Totales',
                    value: '1,247',
                    icon: Icons.people,
                    color: AdminColors.primary,
                    subtitle: '+12% vs mes anterior',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Usuarios Activos',
                    value: '892',
                    icon: Icons.people_alt,
                    color: Colors.green,
                    subtitle: '71.5% del total',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Nuevos Usuarios (mes)',
                    value: '156',
                    icon: Icons.person_add,
                    color: Colors.blue,
                    subtitle: '+8% vs mes anterior',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Tasa de Retención',
                    value: '87%',
                    icon: Icons.trending_up,
                    color: Colors.orange,
                    subtitle: '+3% vs mes anterior',
                  ),
                ),
              ],
            ),

            const SizedBox(height: 24),

            // Métricas financieras - Fila 2
            Row(
              children: [
                Expanded(
                  child: MetricCard(
                    title: 'Ingresos Totales',
                    value: '\$24,589',
                    icon: Icons.attach_money,
                    color: Colors.green[700]!,
                    subtitle: '+18% vs mes anterior',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Ingresos Este Mes',
                    value: '\$8,245',
                    icon: Icons.calendar_today,
                    color: Colors.teal,
                    subtitle: 'Objetivo: \$10,000',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Ticket Promedio',
                    value: '\$29.99',
                    icon: Icons.receipt,
                    color: Colors.indigo,
                    subtitle: '+5% vs mes anterior',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Conversión',
                    value: '3.2%',
                    icon: Icons.show_chart,
                    color: Colors.purple,
                    subtitle: '+0.5% vs mes anterior',
                  ),
                ),
              ],
            ),

            const SizedBox(height: 32),

            // Gráficas y actividad reciente
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Gráfica de usuarios
                Expanded(
                  flex: 2,
                  child: Container(
                    padding: const EdgeInsets.all(24),
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
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            const Text(
                              'Usuarios Nuevos (últimos 7 días)',
                              style: TextStyle(
                                fontSize: 18,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                            IconButton(
                              icon: const Icon(Icons.more_vert),
                              onPressed:
                                  () => Get.snackbar(
                                    'Info',
                                    'Opciones de gráfica',
                                  ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 24),
                        SizedBox(
                          height: 250,
                          child: Center(
                            child: Column(
                              mainAxisAlignment: MainAxisAlignment.center,
                              children: [
                                Icon(
                                  Icons.bar_chart,
                                  size: 80,
                                  color: Colors.grey[300],
                                ),
                                const SizedBox(height: 16),
                                Text(
                                  'Gráfica de usuarios nuevos',
                                  style: TextStyle(color: Colors.grey[600]),
                                ),
                                const SizedBox(height: 8),
                                Text(
                                  '(Integrar librería de gráficas)',
                                  style: TextStyle(
                                    color: Colors.grey[400],
                                    fontSize: 12,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),

                const SizedBox(width: 16),

                // Actividad reciente
                Expanded(
                  child: Container(
                    padding: const EdgeInsets.all(24),
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
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text(
                          'Actividad Reciente',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 24),
                        _buildActivityItem(
                          'Nuevo usuario registrado',
                          'Juan Pérez se registró con Plan Premium',
                          'Hace 5 minutos',
                          Icons.person_add,
                          Colors.green,
                        ),
                        const Divider(height: 32),
                        _buildActivityItem(
                          'Suscripción renovada',
                          'María García renovó Plan Básico',
                          'Hace 1 hora',
                          Icons.refresh,
                          Colors.blue,
                        ),
                        const Divider(height: 32),
                        _buildActivityItem(
                          'Nuevo feedback',
                          'Carlos López dejó una reseña 5⭐',
                          'Hace 2 horas',
                          Icons.star,
                          Colors.orange,
                        ),
                        const Divider(height: 32),
                        _buildActivityItem(
                          'Plan actualizado',
                          'Ana Martínez cambió a Plan Premium',
                          'Hace 4 horas',
                          Icons.arrow_upward,
                          AdminColors.primary,
                        ),
                        const SizedBox(height: 16),
                        Center(
                          child: TextButton(
                            onPressed:
                                () => Get.snackbar(
                                  'Info',
                                  'Ver todas las actividades',
                                ),
                            child: const Text('Ver todas las actividades →'),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),

            const SizedBox(height: 24),

            // Planes más populares
            Container(
              padding: const EdgeInsets.all(24),
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
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        'Planes Más Populares',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      TextButton(
                        onPressed: () => Get.toNamed('/admin/plans'),
                        child: const Text('Ver todos →'),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  Row(
                    children: [
                      Expanded(
                        child: _buildPopularPlan(
                          'Plan Premium',
                          '45%',
                          567,
                          Colors.purple,
                        ),
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: _buildPopularPlan(
                          'Plan Básico',
                          '38%',
                          478,
                          Colors.blue,
                        ),
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: _buildPopularPlan(
                          'Plan Anual',
                          '17%',
                          202,
                          Colors.green,
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildActivityItem(
    String title,
    String description,
    String time,
    IconData icon,
    Color color,
  ) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
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
                title,
                style: const TextStyle(
                  fontWeight: FontWeight.w600,
                  fontSize: 14,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                description,
                style: TextStyle(color: Colors.grey[600], fontSize: 13),
              ),
              const SizedBox(height: 4),
              Text(
                time,
                style: TextStyle(color: Colors.grey[400], fontSize: 12),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildPopularPlan(
    String name,
    String percentage,
    int users,
    Color color,
  ) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.05),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: color.withValues(alpha: 0.2)),
      ),
      child: Column(
        children: [
          Text(
            name,
            style: TextStyle(
              fontWeight: FontWeight.bold,
              fontSize: 16,
              color: color,
            ),
          ),
          const SizedBox(height: 12),
          Text(
            percentage,
            style: const TextStyle(fontSize: 32, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 4),
          Text(
            '$users usuarios',
            style: TextStyle(color: Colors.grey[600], fontSize: 13),
          ),
          const SizedBox(height: 12),
          LinearProgressIndicator(
            value: double.parse(percentage.replaceAll('%', '')) / 100,
            backgroundColor: Colors.grey[200],
            valueColor: AlwaysStoppedAnimation<Color>(color),
            minHeight: 6,
            borderRadius: BorderRadius.circular(3),
          ),
        ],
      ),
    );
  }
}
