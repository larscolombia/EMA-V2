// ignore_for_file: public_member_api_docs, sort_constructors_first
import 'package:ema_educacion_medica_avanzada/app/actions/controllers/actions_list_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_type.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/controllers/quiz_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/app_icons.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ActionsListWidget extends StatelessWidget {
  final _actionsController = Get.find<ActionsListController>();
  final _clinicalCaseController = Get.find<ClinicalCaseController>();
  final _quizController = Get.find<QuizController>();

  ActionsListWidget({super.key,});

  void _onActionSelected(ActionModel action) {
    if (action.type == ActionType.clinicalCase) {
      _clinicalCaseController.showChat(action.itemId);
    }

    if (action.type == ActionType.pdf) {
      // Todo: generar un chat
      // Get.toNamed(Routes.chatDetail.name, arguments: action.itemId);
    }

    if (action.type == ActionType.quizzes) {
      _quizController.useQuiz(action.itemId);
    }
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 0),
      child: Column(
        mainAxisSize: MainAxisSize.max,
        mainAxisAlignment: MainAxisAlignment.start,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.only(bottom: 12, top: 24),
            child: Obx(() => Text(
              _actionsController.typeTitle.value.toUpperCase(),
              style: AppStyles.breadCumb,
            )),
          ),
          
          Divider(),

          Obx(
            () {
              if (_actionsController.actions.isEmpty) {
                return StateMessageWidget(
                  message: 'No se encontraron ${_actionsController.typeTitle}', 
                  type: StateMessageType.noSearchResults,
                );
              }

              return ListView.separated(
                reverse: true,
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: _actionsController.actions.length,
                separatorBuilder: (context, index) => const Divider(),
                itemBuilder: (context, index) {
                  final action = _actionsController.actions[index];

                  return ListTile(
                    contentPadding: const EdgeInsets.only(top: 2, bottom: 2, left: 0, right: 12),
                    title: Text(
                      action.shortTitle,
                      maxLines: 1,
                    ),
                    trailing: AppIcons.arrowRightCircular(
                      height: 24,
                      width: 24,
                      color: AppStyles.tertiaryColor,
                    ),
                    onTap: () {
                      _onActionSelected(action);
                    },
                  );
                },
              );
            },
          ),
        ],
      ),
    );
  }
}
