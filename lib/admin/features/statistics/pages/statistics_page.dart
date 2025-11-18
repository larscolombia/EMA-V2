import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/layout/admin_layout.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/cards/metric_card.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class StatisticsPage extends StatelessWidget {
  const StatisticsPage({super.key});

  @override
  Widget build(BuildContext context) {
    return AdminLayout(
      title: 'Estadísticas y Reportes',
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Filtros y exportación
            Row(
              children: [
                Expanded(
                  child: Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(12),
                      boxShadow: [
                        BoxShadow(
                          color: Colors.black.withValues(alpha: 0.05),
                          blurRadius: 10,
                          offset: const Offset(0, 2),
                        ),
                      ],
                    ),
                    child: Row(
                      children: [
                        const Icon(
                          Icons.calendar_today,
                          size: 20,
                          color: Colors.grey,
                        ),
                        const SizedBox(width: 12),
                        const Text(
                          'Rango: ',
                          style: TextStyle(fontWeight: FontWeight.w600),
                        ),
                        DropdownButton<String>(
                          value: 'Últimos 30 días',
                          underline: const SizedBox(),
                          items:
                              [
                                    'Últimos 7 días',
                                    'Últimos 30 días',
                                    'Últimos 90 días',
                                    'Este año',
                                  ]
                                  .map(
                                    (e) => DropdownMenuItem(
                                      value: e,
                                      child: Text(e),
                                    ),
                                  )
                                  .toList(),
                          onChanged:
                              (v) => Get.snackbar('Filtro', 'Cambiado a: $v'),
                        ),
                        const Spacer(),
                        ElevatedButton.icon(
                          onPressed:
                              () => Get.snackbar(
                                'Exportar',
                                'Exportando a Excel...',
                              ),
                          icon: const Icon(Icons.download, size: 18),
                          label: const Text('Exportar Excel'),
                          style: ElevatedButton.styleFrom(
                            backgroundColor: Colors.green,
                            foregroundColor: Colors.white,
                          ),
                        ),
                        const SizedBox(width: 12),
                        ElevatedButton.icon(
                          onPressed:
                              () => Get.snackbar(
                                'Exportar',
                                'Exportando a PDF...',
                              ),
                          icon: const Icon(Icons.picture_as_pdf, size: 18),
                          label: const Text('Exportar PDF'),
                          style: ElevatedButton.styleFrom(
                            backgroundColor: Colors.red[700],
                            foregroundColor: Colors.white,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),

            const SizedBox(height: 24),

            // Métricas generales
            Row(
              children: [
                Expanded(
                  child: MetricCard(
                    title: 'Casos Clínicos Resueltos',
                    value: '3,842',
                    icon: Icons.medical_services,
                    color: AdminColors.primary,
                    subtitle: '+15% vs período anterior',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Cuestionarios Completados',
                    value: '5,621',
                    icon: Icons.quiz,
                    color: Colors.blue,
                    subtitle: '+22% vs período anterior',
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: MetricCard(
                    title: 'Consultas Realizadas',
                    value: '2,147',
                    icon: Icons.chat,
                    color: Colors.green,
                    subtitle: '+8% vs período anterior',
                  ),
                ),
              ],
            ),

            const SizedBox(height: 24),

            // Gráficas principales
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Uso por categoría
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
                          'Uso por Categoría',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 24),
                        _buildCategoryBar('Cardiología', 842, 1200, Colors.red),
                        const SizedBox(height: 16),
                        _buildCategoryBar(
                          'Neurología',
                          721,
                          1200,
                          Colors.purple,
                        ),
                        const SizedBox(height: 16),
                        _buildCategoryBar('Pediatría', 654, 1200, Colors.blue),
                        const SizedBox(height: 16),
                        _buildCategoryBar(
                          'Medicina Interna',
                          589,
                          1200,
                          Colors.green,
                        ),
                        const SizedBox(height: 16),
                        _buildCategoryBar(
                          'Traumatología',
                          467,
                          1200,
                          Colors.orange,
                        ),
                        const SizedBox(height: 16),
                        _buildCategoryBar('Otros', 569, 1200, Colors.grey),
                      ],
                    ),
                  ),
                ),

                const SizedBox(width: 16),

                // Actividad por hora
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
                          'Actividad por Hora del Día',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 24),
                        SizedBox(
                          height: 280,
                          child: Center(
                            child: Column(
                              mainAxisAlignment: MainAxisAlignment.center,
                              children: [
                                Icon(
                                  Icons.show_chart,
                                  size: 80,
                                  color: Colors.grey[300],
                                ),
                                const SizedBox(height: 16),
                                Text(
                                  'Gráfica de actividad por hora',
                                  style: TextStyle(color: Colors.grey[600]),
                                ),
                                const SizedBox(height: 8),
                                Text(
                                  '(Integrar fl_chart o similar)',
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
              ],
            ),

            const SizedBox(height: 24),

            // Top usuarios y contenido más popular
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Top usuarios activos
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
                          'Top 5 Usuarios Más Activos',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 20),
                        _buildTopUser(
                          1,
                          'Dr. Juan Pérez',
                          'Plan Premium',
                          847,
                          Colors.amber,
                        ),
                        const Divider(),
                        _buildTopUser(
                          2,
                          'Dra. María García',
                          'Plan Premium',
                          782,
                          Colors.grey,
                        ),
                        const Divider(),
                        _buildTopUser(
                          3,
                          'Dr. Carlos López',
                          'Plan Anual',
                          654,
                          Colors.brown,
                        ),
                        const Divider(),
                        _buildTopUser(
                          4,
                          'Dra. Ana Martínez',
                          'Plan Básico',
                          589,
                          null,
                        ),
                        const Divider(),
                        _buildTopUser(
                          5,
                          'Dr. Luis Rodríguez',
                          'Plan Premium',
                          542,
                          null,
                        ),
                      ],
                    ),
                  ),
                ),

                const SizedBox(width: 16),

                // Contenido más popular
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
                          'Contenido Más Popular',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 20),
                        _buildPopularContent(
                          'Caso: Infarto Agudo de Miocardio',
                          'Cardiología',
                          1247,
                          Icons.medical_services,
                          Colors.red,
                        ),
                        const Divider(),
                        _buildPopularContent(
                          'Cuestionario: Neurología Básica',
                          'Neurología',
                          982,
                          Icons.quiz,
                          Colors.purple,
                        ),
                        const Divider(),
                        _buildPopularContent(
                          'Libro: Atlas de Anatomía',
                          'Recursos',
                          847,
                          Icons.book,
                          Colors.blue,
                        ),
                        const Divider(),
                        _buildPopularContent(
                          'Caso: Meningitis Bacteriana',
                          'Neurología',
                          723,
                          Icons.medical_services,
                          Colors.purple,
                        ),
                        const Divider(),
                        _buildPopularContent(
                          'Cuestionario: ECG Avanzado',
                          'Cardiología',
                          654,
                          Icons.quiz,
                          Colors.red,
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildCategoryBar(String category, int value, int max, Color color) {
    final percentage = (value / max * 100).toStringAsFixed(1);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              category,
              style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
            ),
            Text(
              '$value actividades',
              style: TextStyle(color: Colors.grey[600], fontSize: 13),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Stack(
          children: [
            Container(
              height: 8,
              decoration: BoxDecoration(
                color: Colors.grey[200],
                borderRadius: BorderRadius.circular(4),
              ),
            ),
            FractionallySizedBox(
              widthFactor: value / max,
              child: Container(
                height: 8,
                decoration: BoxDecoration(
                  color: color,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          '$percentage%',
          style: TextStyle(
            color: color,
            fontSize: 12,
            fontWeight: FontWeight.w600,
          ),
        ),
      ],
    );
  }

  Widget _buildTopUser(
    int rank,
    String name,
    String plan,
    int activities,
    Color? medalColor,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: medalColor?.withValues(alpha: 0.1) ?? Colors.grey[100],
              shape: BoxShape.circle,
            ),
            child: Center(
              child:
                  medalColor != null
                      ? Icon(Icons.emoji_events, color: medalColor, size: 20)
                      : Text(
                        '$rank',
                        style: const TextStyle(
                          fontWeight: FontWeight.bold,
                          fontSize: 16,
                        ),
                      ),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  name,
                  style: const TextStyle(
                    fontWeight: FontWeight.w600,
                    fontSize: 14,
                  ),
                ),
                Text(
                  plan,
                  style: TextStyle(color: Colors.grey[600], fontSize: 12),
                ),
              ],
            ),
          ),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: AdminColors.primary.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              '$activities',
              style: TextStyle(
                color: AdminColors.primary,
                fontWeight: FontWeight.bold,
                fontSize: 13,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPopularContent(
    String title,
    String category,
    int views,
    IconData icon,
    Color color,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(10),
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
                    fontSize: 13,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                Text(
                  category,
                  style: TextStyle(color: Colors.grey[600], fontSize: 12),
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Row(
            children: [
              Icon(Icons.visibility, size: 16, color: Colors.grey[400]),
              const SizedBox(width: 4),
              Text(
                '$views',
                style: TextStyle(
                  color: Colors.grey[600],
                  fontWeight: FontWeight.w600,
                  fontSize: 13,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
