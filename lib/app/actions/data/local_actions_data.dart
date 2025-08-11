// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/actions/models/action_model.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';

class LocalActionsData extends ILocalData<ActionModel> {
  static final String _tableName = 'actions_v1';
  @override
  String get tableName => _tableName;
  @override
  String get singular => 'la acción';
  @override
  String get plural => 'las acciones';

  @override
  ActionModel fromApi(Map<String, dynamic> map) {
    // Todo: implement fromApi
    throw UnimplementedError('ActionModel.fromApi() has not been implemented.');
  }

  @override
  ActionModel fromMap(Map<String, dynamic> map) {
    return ActionModel.fromMap(map);
  }

  @override
  Map<String, dynamic> toMap(item) {
    return item.toMap();
  }

  static String sqlInstructionCreateTable() {
    return '''
      CREATE TABLE $_tableName (
        id TEXT PRIMARY KEY,
        userId INTEGER NOT NULL,
        itemId TEXT NOT NULL,
        shortTitle TEXT NOT NULL,
        title TEXT NOT NULL,
        type TEXT NOT NULL, -- ActionType
        createdAt INTEGER NOT NULL,
        updatedAt INTEGER NOT NULL,
        categoryId INTEGER
      );
    ''';
  }
}
