// ignore_for_file: public_member_api_docs, sort_constructors_first
import 'package:ema_educacion_medica_avanzada/app/actions/controllers/actions_drawer_list_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_type.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/controllers/quiz_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/app_icons.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ActionsDrawerListWidget extends StatelessWidget {
  final _actionsController = Get.find<ActionsDrawerListController>();
  final _clinicalCaseController = Get.find<ClinicalCaseController>();
  final _chatController = Get.find<ChatController>();
  final _quizController = Get.find<QuizController>();

  ActionsDrawerListWidget({super.key});

  void _onActionSelected(ActionModel action) {
    if (action.type == ActionType.chat) {
      _chatController.showChat(action.itemId);
    }

    if (action.type == ActionType.clinicalCase) {
      _clinicalCaseController.showChat(action.itemId);
    }

    if (action.type == ActionType.quizzes) {
      _quizController.useQuiz(action.itemId);
    }
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 0),
      child: Obx(() {
        if (_actionsController.actions.isEmpty) {
          return StateMessageWidget(
            message: 'Aquí se mostrarán sus acciones.', 
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
                tileColor: AppStyles.whiteColor,
                contentPadding: const EdgeInsets.only(top: 2, bottom: 2, left: 4, right: 8),
                title: Text(
                  action.shortTitle,
                  maxLines: 1,
                ),
                subtitle: Text(action.type.title),
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
        }
      ),
    );
  }
}
