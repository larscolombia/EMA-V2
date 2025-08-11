import 'package:ema_educacion_medica_avanzada/app/actions/controllers/actions_list_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_type.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class QuizzesMenu extends StatelessWidget {
  final _actionsController = Get.find<ActionsListController>();
  final _navigationService = Get.find<NavigationService>();

  QuizzesMenu({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        OverlayMenuButton(
          title: 'Cuestionarios\nMédicos Generales',
          icon: AppIcons.quizzesGeneral(height: 48, width: 48),
          onPressed: () {
            _navigationService.show(OverlayRoutes.quizzesGeneral);
          }
        ),
        Divider(),
        OverlayMenuButton(
          title: 'Cuestionarios\nEspecialidad Médica',
          icon: AppIcons.quizzesEspecial(height: 48, width: 48),
          onPressed: () {
            _navigationService.show(OverlayRoutes.quizzesSpeciality);
          }
        ),
        Divider(),
        OverlayMenuButton(
          title: 'Historial de\nCuestionarios',
          icon: AppIcons.history(height: 48, width: 48),
          onPressed: () {
            _actionsController.showActionsList(ActionType.quizzes);
          }
        ),
      ],
    );
  }
}
