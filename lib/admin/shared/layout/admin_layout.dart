import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/admin_navbar.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/admin_sidebar.dart';
import 'package:flutter/material.dart';

class AdminLayout extends StatefulWidget {
  final String title;
  final Widget child;
  final List<Widget>? actions;

  const AdminLayout({
    super.key,
    required this.title,
    required this.child,
    this.actions,
  });

  @override
  State<AdminLayout> createState() => _AdminLayoutState();
}

class _AdminLayoutState extends State<AdminLayout> {
  final GlobalKey<ScaffoldState> _scaffoldKey = GlobalKey<ScaffoldState>();
  bool _isSidebarExpanded = true;

  @override
  Widget build(BuildContext context) {
    final isDesktop = MediaQuery.of(context).size.width >= 1024;
    final isTablet =
        MediaQuery.of(context).size.width >= 768 &&
        MediaQuery.of(context).size.width < 1024;

    return Scaffold(
      key: _scaffoldKey,
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
                          ? () => _scaffoldKey.currentState?.openDrawer()
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
