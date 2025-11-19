import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/services/dashboard_service.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/models/timeline_data.dart';

class UsersController extends GetxController {
  final DashboardService _service = DashboardService();

  // Observables
  final isLoading = false.obs;
  final users = <UserListItem>[].obs;
  final currentPage = 1.obs;
  final totalPages = 1.obs;
  final totalUsers = 0.obs;

  // Filters
  final searchQuery = ''.obs;
  final statusFilter = 'all'.obs; // all, active, inactive
  final subscriptionFilter = 'all'.obs; // all, subscribed, free

  @override
  void onInit() {
    super.onInit();

    // Check if there are arguments from navigation
    final args = Get.arguments;
    if (args != null && args is Map<String, dynamic>) {
      if (args['search'] != null) {
        searchQuery.value = args['search'];
      }
      if (args['hasSubscription'] != null) {
        subscriptionFilter.value =
            args['hasSubscription'] ? 'subscribed' : 'free';
      }
    }

    fetchUsers();
  }

  Future<void> fetchUsers({int page = 1}) async {
    try {
      isLoading.value = true;

      // Prepare filters
      String? search = searchQuery.value.isEmpty ? null : searchQuery.value;
      bool? hasSubscription;

      if (subscriptionFilter.value == 'subscribed') {
        hasSubscription = true;
      } else if (subscriptionFilter.value == 'free') {
        hasSubscription = false;
      }

      final result = await _service.getUsersList(
        limit: 20,
        offset: (page - 1) * 20,
        search: search,
        hasSubscription: hasSubscription,
      );

      users.value = result['users'] as List<UserListItem>;
      currentPage.value = page;
      totalUsers.value = result['total'] as int;

      // Calculate total pages
      totalPages.value = (totalUsers.value / 20).ceil();
      if (totalPages.value == 0) totalPages.value = 1;
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudieron cargar los usuarios: $e',
        snackPosition: SnackPosition.BOTTOM,
      );
    } finally {
      isLoading.value = false;
    }
  }

  void applySearch(String query) {
    searchQuery.value = query;
    fetchUsers();
  }

  void applyStatusFilter(String status) {
    statusFilter.value = status;
    fetchUsers();
  }

  void applySubscriptionFilter(String filter) {
    subscriptionFilter.value = filter;
    fetchUsers();
  }

  void resetFilters() {
    searchQuery.value = '';
    statusFilter.value = 'all';
    subscriptionFilter.value = 'all';
    fetchUsers();
  }

  void goToPage(int page) {
    if (page >= 1 && page <= totalPages.value) {
      fetchUsers(page: page);
    }
  }

  void nextPage() {
    if (currentPage.value < totalPages.value) {
      goToPage(currentPage.value + 1);
    }
  }

  void previousPage() {
    if (currentPage.value > 1) {
      goToPage(currentPage.value - 1);
    }
  }

  Future<void> exportUsers() async {
    try {
      Get.snackbar(
        'Exportar',
        'Exportando usuarios a CSV...',
        snackPosition: SnackPosition.BOTTOM,
      );

      // TODO: Implement CSV export
      await Future.delayed(const Duration(seconds: 1));

      Get.snackbar(
        'Éxito',
        'Usuarios exportados correctamente',
        snackPosition: SnackPosition.BOTTOM,
      );
    } catch (e) {
      Get.snackbar(
        'Error',
        'No se pudo exportar: $e',
        snackPosition: SnackPosition.BOTTOM,
      );
    }
  }
}
