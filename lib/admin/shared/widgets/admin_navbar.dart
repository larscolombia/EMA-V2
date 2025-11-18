import 'package:ema_educacion_medica_avanzada/admin/features/auth/controllers/admin_auth_controller.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class AdminNavbar extends StatelessWidget {
  final String title;
  final VoidCallback? onMenuPressed;
  final VoidCallback? onSidebarToggle;
  final List<Widget>? actions;

  const AdminNavbar({
    super.key,
    required this.title,
    this.onMenuPressed,
    this.onSidebarToggle,
    this.actions,
  });

  @override
  Widget build(BuildContext context) {
    // Intentar obtener el controller, pero no fallar si no existe
    AdminAuthController? authController;
    try {
      authController = Get.find<AdminAuthController>();
    } catch (e) {
      // Controller no existe, modo desarrollo sin auth
    }

    return Container(
      height: 64,
      decoration: BoxDecoration(
        color: Colors.white,
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.05),
            blurRadius: 4,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        child: Row(
          children: [
            // Menu button para móvil o toggle para desktop
            if (onMenuPressed != null)
              IconButton(
                icon: const Icon(Icons.menu),
                onPressed: onMenuPressed,
                tooltip: 'Menú',
              )
            else if (onSidebarToggle != null)
              IconButton(
                icon: const Icon(Icons.menu_open),
                onPressed: onSidebarToggle,
                tooltip: 'Contraer/Expandir Sidebar',
              ),

            const SizedBox(width: 16),

            // Título
            Text(
              title,
              style: TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.w600,
                color: AdminColors.textPrimary,
              ),
            ),

            const Spacer(),

            // Acciones personalizadas
            if (actions != null) ...actions!,

            const SizedBox(width: 16),

            // Notificaciones
            IconButton(
              icon: Stack(
                children: [
                  const Icon(Icons.notifications_outlined),
                  Positioned(
                    right: 0,
                    top: 0,
                    child: Container(
                      width: 8,
                      height: 8,
                      decoration: BoxDecoration(
                        color: AdminColors.error,
                        shape: BoxShape.circle,
                      ),
                    ),
                  ),
                ],
              ),
              onPressed: () {
                // TODO: Mostrar notificaciones
              },
              tooltip: 'Notificaciones',
            ),

            const SizedBox(width: 8),

            // Perfil de usuario
            if (authController != null)
              _buildUserProfile(authController)
            else
              _buildDevProfile(),
          ],
        ),
      ),
    );
  }

  Widget _buildUserProfile(AdminAuthController authController) {
    return Obx(() {
      final user = authController.currentUser.value;
      return PopupMenuButton<String>(
        offset: const Offset(0, 50),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              CircleAvatar(
                backgroundColor: AdminColors.primary,
                child: Text(
                  user?.firstName.substring(0, 1).toUpperCase() ?? 'A',
                  style: const TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    user?.fullName ?? 'Administrador',
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                      color: AdminColors.textPrimary,
                    ),
                  ),
                  Text(
                    user?.email ?? '',
                    style: TextStyle(
                      fontSize: 12,
                      color: AdminColors.textSecondary,
                    ),
                  ),
                ],
              ),
              const SizedBox(width: 8),
              const Icon(Icons.arrow_drop_down),
            ],
          ),
        ),
        itemBuilder:
            (context) => [
              PopupMenuItem(
                child: Row(
                  children: [
                    Icon(Icons.person, color: AdminColors.textSecondary),
                    const SizedBox(width: 12),
                    const Text('Mi Perfil'),
                  ],
                ),
                onTap: () {},
              ),
              PopupMenuItem(
                child: Row(
                  children: [
                    Icon(Icons.settings, color: AdminColors.textSecondary),
                    const SizedBox(width: 12),
                    const Text('Configuración'),
                  ],
                ),
                onTap: () {},
              ),
              const PopupMenuDivider(),
              PopupMenuItem(
                child: Row(
                  children: [
                    Icon(Icons.logout, color: AdminColors.error),
                    const SizedBox(width: 12),
                    Text(
                      'Cerrar Sesión',
                      style: TextStyle(color: AdminColors.error),
                    ),
                  ],
                ),
                onTap: () {
                  authController.logout();
                },
              ),
            ],
      );
    });
  }

  Widget _buildDevProfile() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          CircleAvatar(
            backgroundColor: AdminColors.primary,
            child: const Text(
              'D',
              style: TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
          const SizedBox(width: 12),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text(
                'Modo Desarrollo',
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                  color: AdminColors.textPrimary,
                ),
              ),
              Text(
                'dev@ema.com',
                style: TextStyle(
                  fontSize: 12,
                  color: AdminColors.textSecondary,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
