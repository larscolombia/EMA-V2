import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/models/timeline_data.dart';
import 'package:intl/intl.dart';

class TimelineLineChart extends StatelessWidget {
  final List<TimelineDataPoint> dataPoints;
  final TimePeriod period;
  final String title;
  final Color lineColor;
  final double Function(TimelineDataPoint) getValue;
  final String Function(double)? formatValue;

  const TimelineLineChart({
    super.key,
    required this.dataPoints,
    required this.period,
    required this.title,
    required this.lineColor,
    required this.getValue,
    this.formatValue,
  });

  @override
  Widget build(BuildContext context) {
    if (dataPoints.isEmpty) {
      return Container(
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
            Text(
              title,
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 40),
            const Center(
              child: Text(
                'No hay datos disponibles para el período seleccionado',
                style: TextStyle(color: Colors.grey),
              ),
            ),
          ],
        ),
      );
    }

    final spots =
        dataPoints.asMap().entries.map((entry) {
          return FlSpot(entry.key.toDouble(), getValue(entry.value));
        }).toList();

    final maxY = spots.map((s) => s.y).reduce((a, b) => a > b ? a : b);
    final minY = spots.map((s) => s.y).reduce((a, b) => a < b ? a : b);

    // Prevent division by zero when all values are the same
    final yRange = maxY - minY;
    final safeInterval = yRange > 0 ? yRange / 5 : 1.0;

    return Container(
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
          Text(
            title,
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 24),
          SizedBox(
            height: 250,
            child: LineChart(
              LineChartData(
                gridData: FlGridData(
                  show: true,
                  drawVerticalLine: false,
                  horizontalInterval: safeInterval,
                  getDrawingHorizontalLine: (value) {
                    return FlLine(
                      color: Colors.grey.withValues(alpha: 0.2),
                      strokeWidth: 1,
                    );
                  },
                ),
                titlesData: FlTitlesData(
                  show: true,
                  rightTitles: const AxisTitles(
                    sideTitles: SideTitles(showTitles: false),
                  ),
                  topTitles: const AxisTitles(
                    sideTitles: SideTitles(showTitles: false),
                  ),
                  bottomTitles: AxisTitles(
                    sideTitles: SideTitles(
                      showTitles: true,
                      reservedSize: 30,
                      interval: _getBottomInterval(),
                      getTitlesWidget: (value, meta) {
                        if (value.toInt() >= dataPoints.length) {
                          return const Text('');
                        }
                        final point = dataPoints[value.toInt()];
                        return Padding(
                          padding: const EdgeInsets.only(top: 8.0),
                          child: Text(
                            _formatDate(point.date),
                            style: const TextStyle(
                              color: Colors.grey,
                              fontSize: 10,
                            ),
                          ),
                        );
                      },
                    ),
                  ),
                  leftTitles: AxisTitles(
                    sideTitles: SideTitles(
                      showTitles: true,
                      reservedSize: 50,
                      interval: safeInterval,
                      getTitlesWidget: (value, meta) {
                        if (formatValue != null) {
                          return Text(
                            formatValue!(value),
                            style: const TextStyle(
                              color: Colors.grey,
                              fontSize: 12,
                            ),
                          );
                        }
                        return Text(
                          value.toInt().toString(),
                          style: const TextStyle(
                            color: Colors.grey,
                            fontSize: 12,
                          ),
                        );
                      },
                    ),
                  ),
                ),
                borderData: FlBorderData(
                  show: true,
                  border: Border(
                    bottom: BorderSide(
                      color: Colors.grey.withValues(alpha: 0.2),
                    ),
                    left: BorderSide(color: Colors.grey.withValues(alpha: 0.2)),
                  ),
                ),
                minX: 0,
                maxX: (dataPoints.length - 1).toDouble(),
                minY: minY * 0.9,
                maxY: maxY * 1.1,
                lineBarsData: [
                  LineChartBarData(
                    spots: spots,
                    isCurved: true,
                    color: lineColor,
                    barWidth: 3,
                    isStrokeCapRound: true,
                    dotData: FlDotData(
                      show: dataPoints.length <= 31,
                      getDotPainter: (spot, percent, barData, index) {
                        return FlDotCirclePainter(
                          radius: 4,
                          color: lineColor,
                          strokeWidth: 2,
                          strokeColor: Colors.white,
                        );
                      },
                    ),
                    belowBarData: BarAreaData(
                      show: true,
                      color: lineColor.withValues(alpha: 0.1),
                    ),
                  ),
                ],
                lineTouchData: LineTouchData(
                  enabled: true,
                  touchTooltipData: LineTouchTooltipData(
                    getTooltipItems: (touchedSpots) {
                      return touchedSpots.map((spot) {
                        final point = dataPoints[spot.x.toInt()];
                        final value =
                            formatValue != null
                                ? formatValue!(spot.y)
                                : spot.y.toInt().toString();
                        return LineTooltipItem(
                          '${_formatDate(point.date)}\n$value',
                          const TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.bold,
                          ),
                        );
                      }).toList();
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

  double _getBottomInterval() {
    final count = dataPoints.length;
    if (count <= 7) return 1;
    if (count <= 14) return 2;
    if (count <= 31) return 5;
    if (count <= 90) return 15;
    return 30;
  }

  String _formatDate(String date) {
    try {
      switch (period) {
        case TimePeriod.day:
          final parsedDate = DateTime.parse(date);
          return DateFormat('dd/MM').format(parsedDate);
        case TimePeriod.week:
          return 'S$date';
        case TimePeriod.month:
          final parts = date.split('-');
          if (parts.length == 2) {
            return '${parts[1]}/${parts[0].substring(2)}';
          }
          return date;
        case TimePeriod.year:
          return date;
      }
    } catch (e) {
      return date;
    }
  }
}
