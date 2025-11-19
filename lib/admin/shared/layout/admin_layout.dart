import 'package:ema_educacion_medica_avanzada/admin/config/admin_routes.dart';
import 'package:ema_educacion_medica_avanzada/admin/core/services/admin_auth_service.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/admin_navbar.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/admin_sidebar.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class AdminLayout extends StatefulWidget {
  final String title;
  final Widget child;
  final List<Widget>? actions;
  final bool requiresAuth;

  const AdminLayout({
    super.key,
    required this.title,
    required this.child,
    this.actions,
    this.requiresAuth = true,
  });

  @override
  State<AdminLayout> createState() => _AdminLayoutState();
}

class _AdminLayoutState extends State<AdminLayout> {
  bool _isSidebarExpanded = true;
  bool _isCheckingAuth = true;

  @override
  void initState() {
    super.initState();
    if (widget.requiresAuth) {
      _checkAuthentication();
    } else {
      _isCheckingAuth = false;
    }
  }

  Future<void> _checkAuthentication() async {
    try {
      final authService = Get.find<AdminAuthService>();
      final user = await authService.getCurrentUser();

      if (user == null || !user.isSuperAdmin) {
        // Redirigir al login si no está autenticado
        WidgetsBinding.instance.addPostFrameCallback((_) {
          Get.offAllNamed(AdminRoutes.login);
        });
      } else {
        setState(() {
          _isCheckingAuth = false;
        });
      }
    } catch (e) {
      print('❌ [ADMIN LAYOUT] Error checking auth: $e');
      setState(() {
        _isCheckingAuth = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (widget.requiresAuth && _isCheckingAuth) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    final isDesktop = MediaQuery.of(context).size.width >= 1024;
    final isTablet =
        MediaQuery.of(context).size.width >= 768 &&
        MediaQuery.of(context).size.width < 1024;

    return Scaffold(
      drawer:
          !isDesktop
              ? Drawer(child: AdminSidebar(isExpanded: true, onToggle: () {}))
              : null,
      body: Row(
        children: [
          // Sidebar fijo para desktop
          if (isDesktop)
            AdminSidebar(
              isExpanded: _isSidebarExpanded,
              onToggle: () {
                setState(() {
                  _isSidebarExpanded = !_isSidebarExpanded;
                });
              },
            ),

          // Contenido principal
          Expanded(
            child: Column(
              children: [
                // Navbar
                AdminNavbar(
                  title: widget.title,
                  onMenuPressed:
                      !isDesktop
                          ? () => Scaffold.of(context).openDrawer()
                          : null,
                  onSidebarToggle:
                      isDesktop
                          ? () {
                            setState(() {
                              _isSidebarExpanded = !_isSidebarExpanded;
                            });
                          }
                          : null,
                  actions: widget.actions,
                ),

                // Área de contenido
                Expanded(
                  child: Container(
                    width: double.infinity,
                    padding: EdgeInsets.all(
                      isDesktop
                          ? 32
                          : isTablet
                          ? 24
                          : 16,
                    ),
                    child: widget.child,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
