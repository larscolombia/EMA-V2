// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/actions/data/local_actions_data.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_type.dart';
import 'package:get/get.dart';

class ActionsService extends GetxService {
  final _localActionData = Get.find<LocalActionsData>();

  Future<void> insertAction(ActionModel action) async {
    await _localActionData.insertOne(action);
  }

  Future<void> updateAction(ActionModel action) async {
    await _localActionData.update(action, 'id = ?', [action.id]);
  }

  Future<void> deleteAction(ActionModel action) async {
    await _localActionData.delete(where: 'id = ?', whereArgs: [action.id]);
  }

  Future<void> deleteActionsByItemId(ActionType type, String itemId) async {
    await _localActionData.delete(
      where: 'type = ? AND itemId = ?',
      whereArgs: [type.toString(), itemId],
    );
  }

  // Siempre incluir el userId, se pide consulta desde userService
  Future<List<ActionModel>> getActions({
    String? where,
    List<Object>? whereArgs,
    int page = 1,
  }) async {
    return _localActionData.getItems(
      where: where,
      whereArgs: whereArgs,
      page: page,
    );
  }
}
