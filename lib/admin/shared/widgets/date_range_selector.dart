import 'package:ema_educacion_medica_avanzada/admin/core/models/timeline_data.dart';
import 'package:flutter/material.dart';

class DateRangeSelector extends StatelessWidget {
  final TimePeriod selectedPeriod;
  final DateTimeRange? customRange;
  final Function(TimePeriod) onPeriodChanged;
  final Function(DateTimeRange) onCustomRangeSelected;
  final VoidCallback? onResetRange;

  const DateRangeSelector({
    super.key,
    required this.selectedPeriod,
    required this.onPeriodChanged,
    required this.onCustomRangeSelected,
    this.customRange,
    this.onResetRange,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
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
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.calendar_month, size: 20, color: Colors.grey),
              const SizedBox(width: 8),
              const Text(
                'Período',
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                  color: Colors.grey,
                ),
              ),
              const Spacer(),
              if (customRange != null && onResetRange != null)
                TextButton.icon(
                  onPressed: onResetRange,
                  icon: const Icon(Icons.clear, size: 16),
                  label: const Text('Limpiar'),
                  style: TextButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 8),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              ...TimePeriod.values.map((period) {
                final isSelected =
                    selectedPeriod == period && customRange == null;
                return _PeriodChip(
                  label: period.label,
                  isSelected: isSelected,
                  onTap: () => onPeriodChanged(period),
                );
              }),
              _PeriodChip(
                label:
                    customRange != null
                        ? 'Personalizado: ${_formatDateRange(customRange!)}'
                        : 'Personalizado',
                isSelected: customRange != null,
                icon: Icons.date_range,
                onTap: () => _showDateRangePicker(context),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Future<void> _showDateRangePicker(BuildContext context) async {
    final DateTimeRange? picked = await showDateRangePicker(
      context: context,
      firstDate: DateTime(2020),
      lastDate: DateTime.now(),
      initialDateRange:
          customRange ??
          DateTimeRange(
            start: DateTime.now().subtract(const Duration(days: 30)),
            end: DateTime.now(),
          ),
      builder: (context, child) {
        return Theme(
          data: Theme.of(context).copyWith(
            colorScheme: ColorScheme.light(
              primary: Theme.of(context).primaryColor,
            ),
          ),
          child: child!,
        );
      },
    );

    if (picked != null) {
      onCustomRangeSelected(picked);
    }
  }

  String _formatDateRange(DateTimeRange range) {
    final start = '${range.start.day}/${range.start.month}';
    final end = '${range.end.day}/${range.end.month}';
    return '$start - $end';
  }
}

class _PeriodChip extends StatelessWidget {
  final String label;
  final bool isSelected;
  final IconData? icon;
  final VoidCallback onTap;

  const _PeriodChip({
    required this.label,
    required this.isSelected,
    required this.onTap,
    this.icon,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(20),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        decoration: BoxDecoration(
          color:
              isSelected
                  ? Theme.of(context).primaryColor
                  : Colors.grey.shade100,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(
            color:
                isSelected
                    ? Theme.of(context).primaryColor
                    : Colors.transparent,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (icon != null) ...[
              Icon(
                icon,
                size: 16,
                color: isSelected ? Colors.white : Colors.grey.shade700,
              ),
              const SizedBox(width: 6),
            ],
            Text(
              label,
              style: TextStyle(
                color: isSelected ? Colors.white : Colors.grey.shade700,
                fontWeight: isSelected ? FontWeight.w600 : FontWeight.normal,
                fontSize: 13,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
