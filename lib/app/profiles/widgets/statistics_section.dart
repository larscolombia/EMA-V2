import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/utils/mock_statistics_data.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class StatisticsSection extends StatelessWidget {
  StatisticsSection({super.key});

  TextStyle _titleStyle(BuildContext context) => Theme.of(context)
      .textTheme
      .titleLarge!
      .copyWith(color: AppStyles.primaryColor, fontWeight: FontWeight.bold);

  TextStyle _labelStyle(BuildContext context) => Theme.of(
    context,
  ).textTheme.bodySmall!.copyWith(color: AppStyles.primary900);

  TextStyle _countStyle(BuildContext context) => Theme.of(context)
      .textTheme
      .headlineMedium!
      .copyWith(color: AppStyles.primary900, fontWeight: FontWeight.bold);

  TextStyle _sectionTitleStyle(BuildContext context) => Theme.of(context)
      .textTheme
      .titleMedium!
      .copyWith(color: AppStyles.primaryColor, fontWeight: FontWeight.bold);

  final UserTestProgressController progressController =
      Get.find<UserTestProgressController>();

  @override
  Widget build(BuildContext context) {
    final ProfileController profileController = Get.find<ProfileController>();
    final hasStatistics =
        profileController.currentProfile.value.activeSubscription?.statistics ==
        1;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 28),
      child: Column(
        children: [
          Align(
            alignment: Alignment.centerLeft,
            child: Text('TUS ESTADÍSTICAS', style: _titleStyle(context)),
          ),
          const SizedBox(height: 16),
          if (hasStatistics)
            Obx(() {
              final profileController = Get.find<ProfileController>();
              final sub =
                  profileController.currentProfile.value.activeSubscription;
              final totalChatsQuota = sub?.consultations ?? 0;
              final totalTestsQuota = sub?.questionnaires ?? 0;
              final totalClinicalQuota = sub?.clinicalCases ?? 0;
              return Column(
                children: [
                  _buildStatisticCard(
                    context,
                    'Chats',
                    progressController.totalChats.value,
                    AppIcons.chats(height: 32, width: 32),
                    total: totalChatsQuota,
                  ),
                  const SizedBox(height: 12),
                  _buildStatisticCard(
                    context,
                    'Cuestionarios',
                    progressController.totalTests.value,
                    AppIcons.quizzesGeneral(height: 32, width: 32),
                    total: totalTestsQuota,
                  ),
                  const SizedBox(height: 12),
                  _buildStatisticCard(
                    context,
                    'Casos Clínicos',
                    progressController.totalClinicalCases.value,
                    AppIcons.clinicalCaseAnalytical(height: 32, width: 32),
                    total: totalClinicalQuota,
                  ),
                ],
              );
            })
          else
            Row(
              children:
                  MockStatisticsData.basicStatistics.map((stat) {
                    return Expanded(
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 4),
                        child: _buildStatisticCard(
                          context,
                          stat['label'],
                          stat['count'],
                          stat['label'] == 'Chats'
                              ? AppIcons.chats(height: 32, width: 32)
                              : stat['label'] == 'Cuestionarios'
                              ? AppIcons.quizzesGeneral(height: 32, width: 32)
                              : AppIcons.clinicalCaseAnalytical(
                                height: 32,
                                width: 32,
                              ),
                        ),
                      ),
                    );
                  }).toList(),
            ),
          const SizedBox(height: 16),
          _buildStaticPointsWidget(context, hasStatistics),
          const SizedBox(height: 16),
          _buildStaticPointsBarChart(context, hasStatistics),
          const SizedBox(height: 24),
          _buildTopCategories(context, hasStatistics),
        ],
      ),
    );
  }

  Widget _buildStaticPointsWidget(BuildContext context, bool isPremium) {
    if (isPremium) {
      return Obx(() {
        if (progressController.isLoadingTestScores.value) {
          return const Center(child: CircularProgressIndicator());
        }
        final totalEarned = progressController.totalScore.value;
        final totalPossible = progressController.totalMaxScore.value;

        return Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: AppStyles.tertiaryColor.withAlpha((0.1 * 255).toInt()),
            borderRadius: BorderRadius.circular(12),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Puntos conseguidos',
                    style: _sectionTitleStyle(context),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '$totalEarned / $totalPossible',
                    style: _countStyle(context),
                  ),
                ],
              ),
              Icon(Icons.emoji_events, color: Colors.yellow[700], size: 36),
            ],
          ),
        );
      });
    } else {
      final points = MockStatisticsData.basicPoints;
      return Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: AppStyles.tertiaryColor.withAlpha((0.1 * 255).toInt()),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Puntos conseguidos', style: _sectionTitleStyle(context)),
                const SizedBox(height: 4),
                Text(
                  '${points['earned']} / ${points['possible']}',
                  style: _countStyle(context),
                ),
              ],
            ),
            Icon(Icons.emoji_events, color: Colors.yellow[700], size: 36),
          ],
        ),
      );
    }
  }

  Widget _buildStaticPointsBarChart(BuildContext context, bool isPremium) {
    if (isPremium) {
      return Obx(() {
        if (progressController.isLoadingMonthlyScores.value) {
          return const Center(child: CircularProgressIndicator());
        }

        if (progressController.monthlyScores.isEmpty) {
          return Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: AppStyles.grey220.withAlpha((0.5 * 255).toInt()),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Puntos por mes', style: _sectionTitleStyle(context)),
                const SizedBox(height: 16),
                Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        Icons.insights,
                        size: 40,
                        color: AppStyles.primaryColor,
                      ),
                      const SizedBox(height: 8),
                      Text(
                        '¡Aún no hay datos de tu progreso!',
                        style: _labelStyle(context).copyWith(fontSize: 14),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'Completa algunas actividades para ver tus estadísticas.',
                        style: _labelStyle(context).copyWith(fontSize: 14),
                        textAlign: TextAlign.center,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          );
        }

        // Ahora, dado que MonthlyScore tiene 'mes' (String "2024-11") y 'puntos' (int)
        final List<MonthlyScore> dynamicScores =
            progressController.monthlyScores.toList();

        // Extraemos el número de mes de la cadena "2024-11" y los puntos obtenidos
        final monthNumbers =
            dynamicScores.map((score) {
              final parts = score.mes.split('-');
              return parts.length == 2 ? int.tryParse(parts[1]) ?? 1 : 1;
            }).toList();
        final puntosPorMes =
            dynamicScores.map((score) => score.puntos).toList();

        return _buildPointsBarChart(context, puntosPorMes, monthNumbers);
      });
    } else {
      final monthlyPoints = MockStatisticsData.basicMonthlyPoints;
      return _buildPointsBarChart(context, monthlyPoints);
    }
  }

  Widget _buildPointsBarChart(
    BuildContext context,
    List<int> pointsByMonth, [
    List<int>? monthNumbers,
  ]) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppStyles.grey220.withAlpha((0.5 * 255).toInt()),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Puntos por mes', style: _sectionTitleStyle(context)),
          const SizedBox(height: 16),
          AspectRatio(
            aspectRatio: 1.5,
            child: BarChart(
              BarChartData(
                alignment: BarChartAlignment.spaceAround,
                maxY:
                    (pointsByMonth.isNotEmpty
                            ? pointsByMonth.reduce((a, b) => a > b ? a : b) *
                                1.2
                            : 250)
                        .toDouble(),
                barGroups: List.generate(
                  pointsByMonth.length,
                  (index) => BarChartGroupData(
                    x: index,
                    barRods: [
                      BarChartRodData(
                        toY: pointsByMonth[index].toDouble(),
                        color: AppStyles.primaryColor,
                        width: 14,
                        borderRadius: const BorderRadius.all(
                          Radius.circular(4),
                        ),
                      ),
                    ],
                  ),
                ),
                titlesData: FlTitlesData(
                  show: true,
                  topTitles: AxisTitles(
                    sideTitles: SideTitles(showTitles: false),
                  ),
                  rightTitles: AxisTitles(
                    sideTitles: SideTitles(showTitles: false),
                  ),
                  leftTitles: AxisTitles(
                    sideTitles: SideTitles(
                      showTitles: true,
                      reservedSize: 40,
                      interval: 50,
                      getTitlesWidget: (value, meta) {
                        if (value == meta.max || value == 0) {
                          return const SizedBox();
                        }
                        return Container(
                          padding: const EdgeInsets.only(right: 8),
                          child: Text(
                            value.toInt().toString(),
                            style: _labelStyle(context).copyWith(fontSize: 12),
                          ),
                        );
                      },
                    ),
                  ),
                  bottomTitles: AxisTitles(
                    sideTitles: SideTitles(
                      showTitles: true,
                      reservedSize: 40,
                      interval: 1,
                      getTitlesWidget: (value, meta) {
                        final monthNumber =
                            monthNumbers != null &&
                                    monthNumbers.length > value.toInt()
                                ? monthNumbers[value.toInt()]
                                : value.toInt() + 1;
                        final monthLetters = [
                          'E',
                          'F',
                          'M',
                          'A',
                          'M',
                          'J',
                          'J',
                          'A',
                          'S',
                          'O',
                          'N',
                          'D',
                        ];
                        return Column(
                          children: [
                            Text(
                              monthNumber.toString(),
                              style: _labelStyle(context).copyWith(
                                fontSize: 10,
                                color: AppStyles.primary900.withValues(
                                  alpha: 0.5,
                                ),
                              ),
                            ),
                            const SizedBox(height: 2),
                            Text(
                              monthLetters[monthNumber - 1],
                              style: _labelStyle(
                                context,
                              ).copyWith(fontSize: 12),
                            ),
                          ],
                        );
                      },
                    ),
                  ),
                ),
                gridData: FlGridData(
                  show: true,
                  drawVerticalLine: false,
                  horizontalInterval: 50,
                  checkToShowHorizontalLine: (value) => value % 50 == 0,
                  getDrawingHorizontalLine:
                      (value) => FlLine(
                        color: AppStyles.grey220,
                        strokeWidth: 1,
                        dashArray: [5, 5],
                      ),
                ),
                borderData: FlBorderData(
                  show: true,
                  border: Border(
                    bottom: BorderSide(color: AppStyles.grey220, width: 1),
                    left: BorderSide(color: AppStyles.grey220, width: 1),
                  ),
                ),
                barTouchData: BarTouchData(
                  enabled: true,
                  handleBuiltInTouches: true,
                  touchTooltipData: BarTouchTooltipData(
                    tooltipPadding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 6,
                    ),
                    tooltipMargin: 8,
                    getTooltipItem: (group, groupIndex, rod, rodIndex) {
                      return BarTooltipItem(
                        rod.toY.toInt().toString(),
                        const TextStyle(
                          color: Colors.white,
                          fontWeight: FontWeight.bold,
                        ),
                      );
                    },
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  IconData _getCategoryIcon(String categoryName) {
    final lower = categoryName.toLowerCase();
    if (lower.contains('cardio')) return Icons.favorite;
    if (lower.contains('pedia')) return Icons.child_care;
    if (lower.contains('odonto')) return Icons.medical_services;
    if (lower.contains('fisio')) return Icons.accessibility_new;
    if (lower.contains('medicina')) return Icons.health_and_safety;
    return Icons.local_hospital;
  }

  Color _getCategoryColor(String categoryName) {
    final lower = categoryName.toLowerCase();
    if (lower.contains('cardio')) return Colors.red;
    if (lower.contains('pedia')) return Colors.pink;
    if (lower.contains('odonto')) return Colors.teal;
    if (lower.contains('fisio')) return Colors.green;
    if (lower.contains('medicina')) return Colors.blue;
    return AppStyles.primaryColor;
  }

  Widget _buildTopCategories(BuildContext context, bool isPremium) {
    if (isPremium) {
      return Obx(() {
        final progressController = Get.find<UserTestProgressController>();
        if (progressController.isLoadingMostStudiedCategory.value) {
          return const Center(child: CircularProgressIndicator());
        }
        final category = progressController.mostStudiedCategory.value;
        if (category == null) {
          return Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: AppStyles.secondaryColor.withAlpha((0.1 * 255).toInt()),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Categoría más estudiada',
                  style: _sectionTitleStyle(context),
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    CircleAvatar(
                      backgroundColor: Colors.lightGreen,
                      child: const Icon(Icons.school, color: Colors.white),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        '¡Empieza a estudiar para ver tu categoría top!',
                        style: _labelStyle(
                          context,
                        ).copyWith(fontSize: 16, fontWeight: FontWeight.bold),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          );
        }
        final iconData = _getCategoryIcon(category.categoryName);
        final iconColor = _getCategoryColor(category.categoryName);
        return Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: AppStyles.secondaryColor.withAlpha((0.1 * 255).toInt()),
            borderRadius: BorderRadius.circular(12),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Categoría más estudiada',
                style: _sectionTitleStyle(context),
              ),
              const SizedBox(height: 16),
              Row(
                children: [
                  CircleAvatar(
                    backgroundColor: iconColor,
                    child: Icon(iconData, color: Colors.white),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          category.categoryName,
                          style: _labelStyle(
                            context,
                          ).copyWith(fontSize: 16, fontWeight: FontWeight.bold),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          '${category.studyCount} test${category.studyCount != 1 ? 's' : ''} realizado${category.studyCount != 1 ? 's' : ''}',
                          style: _labelStyle(context).copyWith(
                            fontSize: 13,
                            color: AppStyles.primary900.withValues(alpha: 0.7),
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ],
          ),
        );
      });
    } else {
      final categories = MockStatisticsData.basicCategories;
      return Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: AppStyles.secondaryColor.withAlpha((0.1 * 255).toInt()),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Categorías más estudiadas',
              style: _sectionTitleStyle(context),
            ),
            const SizedBox(height: 16),
            ListView.builder(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: categories.length,
              itemBuilder: (context, index) {
                final category = categories[index];
                return Container(
                  margin: const EdgeInsets.symmetric(vertical: 8),
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: (category['color'] as Color).withAlpha(
                      (0.1 * 255).toInt(),
                    ),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Row(
                    children: [
                      CircleAvatar(
                        backgroundColor: category['color'] as Color,
                        child: Icon(
                          category['icon'] as IconData,
                          color: Colors.white,
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          category['name'] as String,
                          style: _labelStyle(
                            context,
                          ).copyWith(fontSize: 16, fontWeight: FontWeight.bold),
                        ),
                      ),
                      Text(
                        '${index + 1}°',
                        style: _sectionTitleStyle(
                          context,
                        ).copyWith(fontSize: 14, fontWeight: FontWeight.bold),
                      ),
                    ],
                  ),
                );
              },
            ),
          ],
        ),
      );
    }
  }

  Widget _buildStatisticCard(
    BuildContext context,
    String label,
    int count,
    Widget icon, {
    int? total,
  }) {
    final showTotal = total != null && total > 0;

    // IMPORTANTE: 'total' ya representa los DISPONIBLES que vienen del backend
    // en active_subscription.consultations/questionnaires/clinical_cases
    // NO es el total del plan, sino los que quedan
    final remaining = showTotal ? total : 0;

    final double progress =
        showTotal && total != 0
            ? (count / (count + total)).clamp(0.0, 1.0).toDouble()
            : 0.0;

    // Calcular porcentaje para mostrar (basado en usados vs total real)
    final totalReal = showTotal ? count + total : 0;
    final percentage =
        showTotal && totalReal > 0 ? ((count / totalReal) * 100).round() : 0;

    // Determinar color basado en el porcentaje de uso
    Color progressColor;
    if (percentage >= 90) {
      progressColor = Colors.red.shade600; // Crítico
    } else if (percentage >= 70) {
      progressColor = Colors.orange.shade600; // Advertencia
    } else if (percentage >= 50) {
      progressColor = Colors.amber.shade600; // Moderado
    } else {
      progressColor = AppStyles.primaryColor; // Normal
    }

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppStyles.grey220.withAlpha((0.5 * 255).toInt()),
        borderRadius: BorderRadius.circular(12),
        border:
            showTotal && percentage >= 90
                ? Border.all(color: Colors.red.shade300, width: 2)
                : null,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              icon,
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  label,
                  style: _labelStyle(
                    context,
                  ).copyWith(fontSize: 14, fontWeight: FontWeight.w600),
                ),
              ),
              if (showTotal)
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: progressColor.withAlpha((0.15 * 255).toInt()),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    '$percentage%',
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.bold,
                      color: progressColor,
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          if (!showTotal) ...[
            Text('$count', style: _countStyle(context).copyWith(fontSize: 24)),
          ] else ...[
            // Barra de progreso
            ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: LinearProgressIndicator(
                value: progress.isNaN ? 0.0 : progress,
                minHeight: 8,
                backgroundColor: AppStyles.grey200,
                color: progressColor,
              ),
            ),
            const SizedBox(height: 8),
            // Disponibles a la derecha
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                if (remaining <= 5 && remaining > 0)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 2,
                      ),
                      decoration: BoxDecoration(
                        color: Colors.orange.shade100,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(
                            Icons.warning_amber_rounded,
                            size: 12,
                            color: Colors.orange.shade700,
                          ),
                          const SizedBox(width: 4),
                          Text(
                            'Quedan pocos',
                            style: TextStyle(
                              fontSize: 10,
                              color: Colors.orange.shade700,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ],
                      ),
                    ),
                  )
                else if (remaining == 0)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 2,
                      ),
                      decoration: BoxDecoration(
                        color: Colors.red.shade100,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(
                            Icons.block,
                            size: 12,
                            color: Colors.red.shade700,
                          ),
                          const SizedBox(width: 4),
                          Text(
                            'Agotado',
                            style: TextStyle(
                              fontSize: 10,
                              color: Colors.red.shade700,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                Text(
                  'Disponibles: $remaining',
                  style: _labelStyle(context).copyWith(
                    fontSize: 11,
                    color: AppStyles.primary900.withValues(alpha: 0.7),
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}
