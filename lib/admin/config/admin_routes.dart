import 'package:ema_educacion_medica_avanzada/admin/config/admin_bindings.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/auth/pages/admin_login_page.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/dashboard/pages/dashboard_page.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/plans/pages/plans_page.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/books/pages/books_page.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/users/pages/users_page.dart';
import 'package:get/get.dart';

class AdminRoutes {
  static const String login = '/admin/login';
  static const String dashboard = '/admin/dashboard';
  static const String plans = '/admin/plans';
  static const String books = '/admin/books';
  static const String users = '/admin/users';

  static List<GetPage> routes = [
    GetPage(
      name: login,
      page: () => const AdminLoginPage(),
      binding: AdminBindings(),
    ),
    GetPage(
      name: dashboard,
      page: () => const DashboardPage(),
      binding: AdminBindings(),
    ),
    GetPage(
      name: plans,
      page: () => const PlansPage(),
      binding: AdminBindings(),
    ),
    GetPage(
      name: books,
      page: () => const BooksPage(),
      binding: AdminBindings(),
    ),
    GetPage(
      name: users,
      page: () => const UsersPage(),
      binding: AdminBindings(),
    ),
  ];
}
