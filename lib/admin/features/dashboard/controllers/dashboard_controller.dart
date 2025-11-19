import 'package:ema_educacion_medica_avanzada/admin/core/models/dashboard_stats.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/models/timeline_data.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/services/dashboard_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class DashboardController extends GetxController {
  final DashboardService _service = DashboardService();

  // Observables
  final Rx<DashboardStats?> stats = Rx<DashboardStats?>(null);
  final Rx<TimelineData?> timeline = Rx<TimelineData?>(null);
  final isLoading = false.obs;
  final isLoadingTimeline = false.obs;
  final error = ''.obs;

  // Filtros
  final selectedPeriod = TimePeriod.month.obs;
  final Rx<DateTimeRange?> selectedDateRange = Rx<DateTimeRange?>(null);

  @override
  void onInit() {
    super.onInit();
    fetchStats();
    fetchTimeline();
  }

  Future<void> fetchStats() async {
    try {
      isLoading.value = true;
      error.value = '';
      final result = await _service.getStats();
      stats.value = result;
    } catch (e) {
      error.value = e.toString();
      Get.snackbar(
        'Error',
        'No se pudieron cargar las estadísticas del dashboard',
        snackPosition: SnackPosition.BOTTOM,
      );
    } finally {
      isLoading.value = false;
    }
  }

  Future<void> fetchTimeline() async {
    try {
      isLoadingTimeline.value = true;
      final result = await _service.getTimeline(
        period: selectedPeriod.value,
        startDate: selectedDateRange.value?.start,
        endDate: selectedDateRange.value?.end,
      );
      timeline.value = result;
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudieron cargar los datos de timeline',
        snackPosition: SnackPosition.BOTTOM,
      );
    } finally {
      isLoadingTimeline.value = false;
    }
  }

  Future<void> refresh() async {
    await Future.wait([fetchStats(), fetchTimeline()]);
  }

  void setPeriod(TimePeriod period) {
    selectedPeriod.value = period;
    selectedDateRange.value = null; // Reset custom date range
    fetchTimeline();
  }

  void setDateRange(DateTimeRange range) {
    selectedDateRange.value = range;
    fetchTimeline();
  }

  void resetDateRange() {
    selectedDateRange.value = null;
    fetchTimeline();
  }

  // Drill-down methods
  Future<void> showUsersList({String? search, bool? hasSubscription}) async {
    // Navigate to users page with filter arguments
    Get.toNamed(
      '/admin/users',
      arguments: {'search': search, 'hasSubscription': hasSubscription},
    );
  }

  Future<void> showSubscriptionsHistory({
    int? userId,
    int? planId,
    bool activeOnly = false,
  }) async {
    try {
      final result = await _service.getSubscriptionsHistory(
        userId: userId,
        planId: planId,
        activeOnly: activeOnly,
      );

      // Navigate to subscriptions history page or show dialog
      Get.toNamed('/admin/subscriptions/history', arguments: result);
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudo cargar el historial de suscripciones',
        snackPosition: SnackPosition.BOTTOM,
      );
    }
  }

  // Export methods
  Future<void> exportToCSV() async {
    try {
      // TODO: Implement CSV export using csv package
      Get.snackbar(
        'Información',
        'La exportación a CSV estará disponible próximamente',
        snackPosition: SnackPosition.BOTTOM,
      );
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudo exportar a CSV',
        snackPosition: SnackPosition.BOTTOM,
      );
    }
  }
}
