import 'package:ema_educacion_medica_avanzada/app/actions/controllers/actions_list_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_type.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ClinicalCasesMenu extends StatelessWidget {
  final _actionsController = Get.find<ActionsListController>();
  final navigationService = Get.find<NavigationService>();

  ClinicalCasesMenu({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        OverlayMenuButton(
          title: 'Caso Clínico\nAnalítico',
          icon: AppIcons.clinicalCaseAnalytical(height: 48, width: 48),
          onPressed: () {
            navigationService.show(OverlayRoutes.clinicalCaseAnlytical);
          }
        ),

        Divider(),

        OverlayMenuButton(
          title: 'Caso Clínico\nInteractivo',
          icon: AppIcons.clinicalCaseInteractive(height: 48, width: 48),
          onPressed: () {
            navigationService.show(OverlayRoutes.clinicalCaseInteractive);
          }
        ),

        Divider(),

        OverlayMenuButton(
          title: 'Historial de\nCasos Clínicos',
          icon: AppIcons.history(height: 48, width: 48),
          onPressed: () {
            _actionsController.showActionsList(ActionType.clinicalCase);
          }
        ),
      ],
    );
  }
}
