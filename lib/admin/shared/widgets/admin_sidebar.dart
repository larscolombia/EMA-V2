import 'package:ema_educacion_medica_avanzada/admin/config/admin_routes.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class AdminSidebar extends StatelessWidget {
  final bool isExpanded;
  final VoidCallback onToggle;

  const AdminSidebar({
    super.key,
    required this.isExpanded,
    required this.onToggle,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: isExpanded ? 260 : 80,
      decoration: BoxDecoration(
        color: AdminColors.sidebarBackground,
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.1),
            blurRadius: 8,
            offset: const Offset(2, 0),
          ),
        ],
      ),
      child: Column(
        children: [
          // Header/Logo
          Container(
            height: 64,
            padding: const EdgeInsets.symmetric(horizontal: 16),
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
              ),
            ),
            child: Center(
              child:
                  isExpanded
                      ? Image.asset(
                        'assets/images/logotype_white.png',
                        height: 40,
                        fit: BoxFit.contain,
                      )
                      : Image.asset(
                        'assets/images/logo.png',
                        width: 40,
                        height: 40,
                        fit: BoxFit.contain,
                      ),
            ),
          ),

          // Menu Items
          Expanded(
            child: ListView(
              padding: const EdgeInsets.symmetric(vertical: 16),
              children: [
                _buildMenuItem(
                  icon: Icons.dashboard,
                  label: 'Dashboard',
                  route: AdminRoutes.dashboard,
                ),
                const SizedBox(height: 8),
                _buildMenuItem(
                  icon: Icons.people,
                  label: 'Usuarios',
                  route: AdminRoutes.users,
                ),
                _buildMenuItem(
                  icon: Icons.card_membership,
                  label: 'Planes',
                  route: AdminRoutes.plans,
                ),
                _buildMenuItem(
                  icon: Icons.book,
                  label: 'Libros',
                  route: AdminRoutes.books,
                ),
                _buildMenuItem(
                  icon: Icons.bar_chart,
                  label: 'Estadísticas',
                  route: AdminRoutes.statistics,
                ),
              ],
            ),
          ),

          // Footer
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              border: Border(
                top: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
              ),
            ),
            child:
                isExpanded
                    ? Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'EMA Admin v1.0',
                          style: TextStyle(
                            color: AdminColors.sidebarText.withValues(
                              alpha: 0.6,
                            ),
                            fontSize: 12,
                          ),
                        ),
                        Text(
                          '© 2025',
                          style: TextStyle(
                            color: AdminColors.sidebarText.withValues(
                              alpha: 0.6,
                            ),
                            fontSize: 11,
                          ),
                        ),
                      ],
                    )
                    : Icon(
                      Icons.info_outline,
                      color: AdminColors.sidebarText.withValues(alpha: 0.6),
                      size: 20,
                    ),
          ),
        ],
      ),
    );
  }

  Widget _buildMenuItem({
    required IconData icon,
    required String label,
    required String route,
  }) {
    final isActive = Get.currentRoute == route;

    return Tooltip(
      message: label,
      child: InkWell(
        onTap: () => Get.toNamed(route),
        child: Container(
          margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
          decoration: BoxDecoration(
            color:
                isActive ? AdminColors.sidebarItemActive : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Icon(
                icon,
                color:
                    isActive
                        ? AdminColors.sidebarTextActive
                        : AdminColors.sidebarText,
                size: 24,
              ),
              if (isExpanded) ...[
                const SizedBox(width: 16),
                Expanded(
                  child: Text(
                    label,
                    style: TextStyle(
                      color:
                          isActive
                              ? AdminColors.sidebarTextActive
                              : AdminColors.sidebarText,
                      fontSize: 15,
                      fontWeight: isActive ? FontWeight.w600 : FontWeight.w400,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
