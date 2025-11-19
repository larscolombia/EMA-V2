import 'package:ema_educacion_medica_avanzada/admin/features/dashboard/controllers/dashboard_controller.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/layout/admin_layout.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/cards/metric_card.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/charts/line_chart_widget.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/date_range_selector.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class DashboardPage extends StatelessWidget {
  const DashboardPage({super.key});

  @override
  Widget build(BuildContext context) {
    final controller = Get.put(DashboardController());

    return AdminLayout(
      title: 'Dashboard',
      actions: [
        IconButton(
          icon: const Icon(Icons.refresh),
          tooltip: 'Actualizar',
          onPressed: controller.refresh,
        ),
      ],
      child: Obx(() {
        if (controller.isLoading.value && controller.stats.value == null) {
          return const Center(child: CircularProgressIndicator());
        }

        if (controller.error.value.isNotEmpty &&
            controller.stats.value == null) {
          return Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const Icon(Icons.error_outline, size: 64, color: Colors.red),
                const SizedBox(height: 16),
                Text(
                  'Error al cargar estadísticas',
                  style: const TextStyle(fontSize: 18),
                ),
                const SizedBox(height: 8),
                Text(
                  controller.error.value,
                  style: TextStyle(color: Colors.grey[600]),
                ),
                const SizedBox(height: 16),
                ElevatedButton(
                  onPressed: controller.refresh,
                  child: const Text('Reintentar'),
                ),
              ],
            ),
          );
        }

        final stats = controller.stats.value;
        if (stats == null) {
          return const Center(child: Text('No hay datos disponibles'));
        }

        return RefreshIndicator(
          onRefresh: controller.refresh,
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            physics: const AlwaysScrollableScrollPhysics(),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Métricas principales - Fila 1: Usuarios
                Row(
                  children: [
                    Expanded(
                      child: MetricCard(
                        title: 'Usuarios Totales',
                        value: '${stats.users.total}',
                        icon: Icons.people,
                        color: AdminColors.primary,
                        subtitle: '${stats.users.active} activos',
                        onTap: () => controller.showUsersList(),
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: MetricCard(
                        title: 'Usuarios Activos',
                        value: '${stats.users.active}',
                        icon: Icons.people_alt,
                        color: Colors.green,
                        subtitle:
                            '${stats.users.retentionRate.toStringAsFixed(1)}% del total',
                        onTap:
                            () =>
                                controller.showUsersList(hasSubscription: true),
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: MetricCard(
                        title: 'Nuevos Usuarios (mes)',
                        value: '${stats.users.newThisMonth}',
                        icon: Icons.person_add,
                        color: Colors.blue,
                        subtitle:
                            stats.users.growthPercent >= 0
                                ? '+${stats.users.growthPercent.toStringAsFixed(1)}% vs mes anterior'
                                : '${stats.users.growthPercent.toStringAsFixed(1)}% vs mes anterior',
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: MetricCard(
                        title: 'Tasa de Retención',
                        value:
                            '${stats.users.retentionRate.toStringAsFixed(0)}%',
                        icon: Icons.trending_up,
                        color: Colors.orange,
                        subtitle: 'Usuarios con suscripción activa',
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
                        value:
                            '\$${stats.financial.totalRevenue.toStringAsFixed(2)}',
                        icon: Icons.attach_money,
                        color: Colors.green[700]!,
                        subtitle: 'Desde el inicio',
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: MetricCard(
                        title: 'Ingresos Este Mes',
                        value:
                            '\$${stats.financial.monthlyRevenue.toStringAsFixed(2)}',
                        icon: Icons.calendar_today,
                        color: Colors.teal,
                        subtitle:
                            stats.financial.growthPercent >= 0
                                ? '+${stats.financial.growthPercent.toStringAsFixed(1)}% vs mes anterior'
                                : '${stats.financial.growthPercent.toStringAsFixed(1)}% vs mes anterior',
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: MetricCard(
                        title: 'Ticket Promedio',
                        value:
                            '\$${stats.financial.averageTicket.toStringAsFixed(2)}',
                        icon: Icons.receipt,
                        color: Colors.indigo,
                        subtitle: 'Por suscripción',
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: MetricCard(
                        title: 'Conversión',
                        value:
                            '${stats.financial.conversionRate.toStringAsFixed(1)}%',
                        icon: Icons.show_chart,
                        color: Colors.purple,
                        subtitle: 'Usuarios de pago',
                      ),
                    ),
                  ],
                ),

                const SizedBox(height: 32),

                // Selector de período
                DateRangeSelector(
                  selectedPeriod: controller.selectedPeriod.value,
                  customRange: controller.selectedDateRange.value,
                  onPeriodChanged: controller.setPeriod,
                  onCustomRangeSelected: controller.setDateRange,
                  onResetRange: controller.resetDateRange,
                ),

                const SizedBox(height: 24),

                // Gráficas
                if (controller.isLoadingTimeline.value)
                  const Center(
                    child: Padding(
                      padding: EdgeInsets.all(48.0),
                      child: CircularProgressIndicator(),
                    ),
                  )
                else if (controller.timeline.value != null) ...[
                  // Gráfica de nuevos usuarios
                  TimelineLineChart(
                    dataPoints: controller.timeline.value!.points,
                    period: controller.selectedPeriod.value,
                    title: 'Nuevos Usuarios',
                    lineColor: Colors.blue,
                    getValue: (point) => point.users.toDouble(),
                  ),

                  const SizedBox(height: 24),

                  // Gráfica de ingresos
                  TimelineLineChart(
                    dataPoints: controller.timeline.value!.points,
                    period: controller.selectedPeriod.value,
                    title: 'Ingresos',
                    lineColor: Colors.green,
                    getValue: (point) => point.revenue,
                    formatValue: (value) => '\$${value.toStringAsFixed(0)}',
                  ),
                ],

                const SizedBox(height: 32),

                // Actividad del sistema
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Actividad
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
                              'Actividad del Sistema',
                              style: TextStyle(
                                fontSize: 18,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                            const SizedBox(height: 24),
                            _buildActivityRow(
                              'Consultas AI',
                              stats.activity.totalConsultations,
                              Icons.chat,
                              Colors.blue,
                            ),
                            const Divider(height: 24),
                            _buildActivityRow(
                              'Tests Generados',
                              stats.activity.totalTests,
                              Icons.quiz,
                              Colors.green,
                            ),
                            const Divider(height: 24),
                            _buildActivityRow(
                              'Casos Clínicos',
                              stats.activity.totalClinicalCases,
                              Icons.medical_services,
                              Colors.red,
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
                            if (stats.recentActivity.isEmpty)
                              const Center(
                                child: Padding(
                                  padding: EdgeInsets.all(32.0),
                                  child: Text(
                                    'No hay actividad reciente',
                                    style: TextStyle(color: Colors.grey),
                                  ),
                                ),
                              )
                            else
                              ...stats.recentActivity
                                  .take(5)
                                  .map(
                                    (activity) => Column(
                                      children: [
                                        _buildActivityItem(
                                          activity.title,
                                          activity.description,
                                          _formatTimestamp(activity.timestamp),
                                          Icons.person_add,
                                          Colors.green,
                                        ),
                                        if (activity !=
                                            stats.recentActivity.take(5).last)
                                          const Divider(height: 32),
                                      ],
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
                      if (stats.plans.isEmpty)
                        const Center(
                          child: Padding(
                            padding: EdgeInsets.all(32.0),
                            child: Text(
                              'No hay planes disponibles',
                              style: TextStyle(color: Colors.grey),
                            ),
                          ),
                        )
                      else
                        Row(
                          children:
                              stats.plans.take(3).map((plan) {
                                final index = stats.plans.indexOf(plan);
                                final colors = [
                                  Colors.purple,
                                  Colors.blue,
                                  Colors.green,
                                ];
                                return Expanded(
                                  child: Padding(
                                    padding: EdgeInsets.only(
                                      right: index < 2 ? 16.0 : 0,
                                    ),
                                    child: _buildPopularPlan(
                                      plan.name,
                                      '${plan.percentage.toStringAsFixed(1)}%',
                                      plan.subscriberCount,
                                      colors[index % colors.length],
                                    ),
                                  ),
                                );
                              }).toList(),
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        );
      }),
    );
  }

  Widget _buildActivityRow(
    String label,
    int count,
    IconData icon,
    Color color,
  ) {
    return Row(
      children: [
        Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Icon(icon, color: color, size: 24),
        ),
        const SizedBox(width: 16),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                label,
                style: const TextStyle(
                  fontWeight: FontWeight.w600,
                  fontSize: 14,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                '$count total',
                style: TextStyle(color: Colors.grey[600], fontSize: 13),
              ),
            ],
          ),
        ),
        Text(
          '$count',
          style: const TextStyle(fontSize: 24, fontWeight: FontWeight.bold),
        ),
      ],
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
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
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
            textAlign: TextAlign.center,
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

  String _formatTimestamp(DateTime timestamp) {
    final now = DateTime.now();
    final difference = now.difference(timestamp);

    if (difference.inMinutes < 1) {
      return 'Hace menos de 1 minuto';
    } else if (difference.inMinutes < 60) {
      return 'Hace ${difference.inMinutes} minuto${difference.inMinutes > 1 ? 's' : ''}';
    } else if (difference.inHours < 24) {
      return 'Hace ${difference.inHours} hora${difference.inHours > 1 ? 's' : ''}';
    } else if (difference.inDays < 30) {
      return 'Hace ${difference.inDays} día${difference.inDays > 1 ? 's' : ''}';
    } else {
      return 'Hace más de un mes';
    }
  }
}
