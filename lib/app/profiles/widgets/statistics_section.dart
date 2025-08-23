import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/utils/mock_statistics_data.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class StatisticsSection extends StatelessWidget {
  StatisticsSection({super.key});

  TextStyle _titleStyle(BuildContext context) =>
      Theme.of(context).textTheme.titleLarge!.copyWith(
            color: AppStyles.primaryColor,
            fontWeight: FontWeight.bold,
          );

  TextStyle _labelStyle(BuildContext context) => Theme.of(context)
      .textTheme
      .bodySmall!
      .copyWith(color: AppStyles.primary900);

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
            child: Text(
              'TUS ESTADÍSTICAS',
              style: _titleStyle(context),
            ),
          ),
          const SizedBox(height: 16),
          if (hasStatistics)
            Obx(() {
              final profileController = Get.find<ProfileController>();
              final sub = profileController.currentProfile.value.activeSubscription;
              final totalChatsQuota = sub?.consultations ?? 0;
              final totalTestsQuota = sub?.questionnaires ?? 0;
              final totalClinicalQuota = sub?.clinicalCases ?? 0;
              return Row(
                children: [
                  Expanded(
                    child: _buildStatisticCard(
                      context,
                      'Chats',
                      progressController.totalChats.value,
                      AppIcons.chats(height: 32, width: 32),
                      total: totalChatsQuota,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: _buildStatisticCard(
                      context,
                      'Cuestionarios',
                      progressController.totalTests.value,
                      AppIcons.quizzesGeneral(height: 32, width: 32),
                      total: totalTestsQuota,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: _buildStatisticCard(
                      context,
                      'Casos Clínicos',
                      progressController.totalClinicalCases.value,
                      AppIcons.clinicalCaseAnalytical(height: 32, width: 32),
                      total: totalClinicalQuota,
                    ),
                  ),
                ],
              );
            })
          else
            Row(
              children: MockStatisticsData.basicStatistics.map((stat) {
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
                                  height: 32, width: 32),
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
        final totalEarned = progressController.totalCorrectas.value;
        final totalPossible = progressController.totalPreguntas.value;

        // print('totalEarned: $totalEarned');
        // print('totalPossible: $totalPossible');

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
                  Text('Puntos conseguidos',
                      style: _sectionTitleStyle(context)),
                  const SizedBox(height: 4),
                  Text('$totalEarned / $totalPossible',
                      style: _countStyle(context)),
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
                Text('${points['earned']} / ${points['possible']}',
                    style: _countStyle(context)),
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
                      Icon(Icons.insights,
                          size: 40, color: AppStyles.primaryColor),
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

        // Ahora, dado que MonthlyScore tiene 'mes' (int) y 'puntos' (int)
        final List<MonthlyScore> dynamicScores =
            progressController.monthlyScores.toList();

        // Extraemos directamente el número de mes y los puntos obtenidos
        final monthNumbers = dynamicScores.map((score) => score.mes).toList();
        final puntosPorMes =
            dynamicScores.map((score) => score.puntos).toList();

        return _buildPointsBarChart(
          context,
          puntosPorMes,
          monthNumbers,
        );
      });
    } else {
      final monthlyPoints = MockStatisticsData.basicMonthlyPoints;
      return _buildPointsBarChart(context, monthlyPoints);
    }
  }

  Widget _buildPointsBarChart(BuildContext context, List<int> pointsByMonth,
      [List<int>? monthNumbers]) {
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
                maxY: (pointsByMonth.isNotEmpty
                        ? pointsByMonth.reduce((a, b) => a > b ? a : b) * 1.2
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
                        borderRadius:
                            const BorderRadius.all(Radius.circular(4)),
                      ),
                    ],
                  ),
                ),
                titlesData: FlTitlesData(
                  show: true,
                  topTitles:
                      AxisTitles(sideTitles: SideTitles(showTitles: false)),
                  rightTitles:
                      AxisTitles(sideTitles: SideTitles(showTitles: false)),
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
                        final monthNumber = monthNumbers != null &&
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
                          'D'
                        ];
                        return Column(
                          children: [
                            Text(
                              monthNumber.toString(),
                              style: _labelStyle(context).copyWith(
                                fontSize: 10,
                                color: AppStyles.primary900.withValues(alpha: 0.5),
                              ),
                            ),
                            const SizedBox(height: 2),
                            Text(
                              monthLetters[monthNumber - 1],
                              style:
                                  _labelStyle(context).copyWith(fontSize: 12),
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
                  getDrawingHorizontalLine: (value) => FlLine(
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
                    tooltipPadding:
                        const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
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
                Text('Categoría más estudiada',
                    style: _sectionTitleStyle(context)),
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
                        style: _labelStyle(context).copyWith(
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                        ),
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
              Text('Categoría más estudiada',
                  style: _sectionTitleStyle(context)),
              const SizedBox(height: 16),
              Row(
                children: [
                  CircleAvatar(
                    backgroundColor: iconColor,
                    child: Icon(iconData, color: Colors.white),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      category.categoryName,
                      style: _labelStyle(context).copyWith(
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                      ),
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
            Text('Categorías más estudiadas',
                style: _sectionTitleStyle(context)),
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
                    color: (category['color'] as Color)
                        .withAlpha((0.1 * 255).toInt()),
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
                          style: _labelStyle(context).copyWith(
                              fontSize: 16, fontWeight: FontWeight.bold),
                        ),
                      ),
                      Text(
                        '${index + 1}°',
                        style: _sectionTitleStyle(context).copyWith(
                            fontSize: 14, fontWeight: FontWeight.bold),
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
      BuildContext context, String label, int count, Widget icon, {
      int? total,
    }) {
  final showTotal = total != null && total > 0;
  final double progress = showTotal && total != 0
    ? (count / total!).clamp(0.0, 1.0).toDouble()
    : 0.0;
    return Container(
      constraints: const BoxConstraints(minHeight: 100),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: AppStyles.grey220.withAlpha((0.5 * 255).toInt()),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              icon,
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  label,
                  style: _labelStyle(context).copyWith(fontSize: 13),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
            Text(
              showTotal ? '$count / $total' : '$count',
              style: _countStyle(context),
            ),
          if (showTotal) ...[
            const SizedBox(height: 6),
            ClipRRect(
              borderRadius: BorderRadius.circular(4),
              child: LinearProgressIndicator(
                value: progress.isNaN ? 0.0 : progress,
                minHeight: 5,
                backgroundColor: AppStyles.grey200,
                color: AppStyles.primaryColor,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
