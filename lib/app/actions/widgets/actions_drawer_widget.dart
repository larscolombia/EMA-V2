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
    return Obx(() {
      if (_actionsController.actions.isEmpty) {
        return Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: StateMessageWidget(
            message: 'Aquí se mostrarán sus acciones.',
            type: StateMessageType.noSearchResults,
          ),
        );
      }
      return ListView.separated(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        reverse: true,
        itemCount: _actionsController.actions.length,
        separatorBuilder: (context, index) => const Divider(height: 8),
        itemBuilder: (context, index) {
          final action = _actionsController.actions[index];
          return ListTile(
            tileColor: AppStyles.whiteColor,
            contentPadding: const EdgeInsets.only(
              top: 0,
              bottom: 0,
              left: 4,
              right: 4,
            ),
            title: Text(
              action.shortTitle,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              softWrap: false,
            ),
            subtitle: Text(
              action.type.title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              softWrap: false,
            ),
            trailing: SizedBox(
              width: 72,
              child: Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  IconButton(
                    tooltip: 'Eliminar',
                    padding: EdgeInsets.zero,
                    constraints: const BoxConstraints(),
                    onPressed: () async {
                      final confirmed = await showDialog<bool>(
                        context: context,
                        builder:
                            (ctx) => AlertDialog(
                              title: const Text('Eliminar'),
                              content: const Text(
                                '¿Deseas eliminar este elemento?',
                              ),
                              actions: [
                                TextButton(
                                  onPressed: () => Navigator.of(ctx).pop(false),
                                  child: const Text('Cancelar'),
                                ),
                                FilledButton(
                                  onPressed: () => Navigator.of(ctx).pop(true),
                                  child: const Text('Eliminar'),
                                ),
                              ],
                            ),
                      );
                      if (confirmed == true) {
                        if (action.type == ActionType.chat) {
                          await _chatController.chatsService.deleteChat(
                            action.itemId,
                          );
                        } else {
                          await _actionsController.deleteAction(action);
                        }
                        _actionsController.loadActions(0);
                      }
                    },
                    icon: const Icon(
                      Icons.delete_outline,
                      color: Colors.redAccent,
                      size: 20,
                    ),
                  ),
                  AppIcons.arrowRightCircular(
                    height: 22,
                    width: 22,
                    color: AppStyles.tertiaryColor,
                  ),
                ],
              ),
            ),
            onTap: () => _onActionSelected(action),
          );
        },
      );
    });
  }
}
