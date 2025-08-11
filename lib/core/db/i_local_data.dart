import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';


abstract class ILocalData<T> {
  final db = Get.find<DatabaseService>().db;
  // static String sqlInstructionCreateTable = '';
  String get tableName;
  String get singular;
  String get plural;

  T fromApi(Map<String, dynamic> map);
  T fromMap(Map<String, dynamic> map);
  Map<String, dynamic> toMap(dynamic item);
  
  Future<void> delete({required String where, required List<Object> whereArgs}) async {
    try {
      await db.delete(tableName, where: where, whereArgs: whereArgs);
    } catch (e) {
      Logger.error(e.toString(), className: 'LocalData', methodName: 'delete', meta: 'tableName: $tableName');
      throw Exception('Error al eliminar $singular del dispositivo.\n${e.toString()}');
    }
  }

  Future<T?> getById(String where, List<Object> whereArgs) async {
    try {
      final items = await getItems(where: where, whereArgs: whereArgs, limit: 1);

      return items.isNotEmpty
        ? items.first
        : null;

    } catch (e) {
      Logger.error(e.toString(), className: 'LocalQuestionsData', methodName: 'getById', meta: 'tableName: $tableName');
      throw Exception('Error al obtener $singular del dispositivo.\n${e.toString()}');
    }
  }

  Future<List<T>> getItems({String? where, List<Object>? whereArgs, String? orderBy, int page = 1, int? limit}) async {
    try {
      String sql;

      if (where != null && whereArgs == null) {
        throw Exception('El parámetro whereArgs es obligatorio cuando se especifica el parámetro where.');
      }

      sql = 'SELECT * FROM $tableName';
      sql += where != null && where.isNotEmpty ? ' WHERE $where' : '';
      sql += orderBy != null && orderBy.isNotEmpty ? ' ORDER BY $orderBy' : '';
      sql += limit != null ? ' LIMIT $limit OFFSET ${(page - 1) * limit};' : '';

      final mapList = (await db.rawQuery(sql, whereArgs)).cast<Map<String, dynamic>>();

      return mapList.map((e) => fromMap(e)).toList();

    } catch (e) {
      Logger.error(e.toString(), className: 'LocalData', methodName: 'getItems', meta: 'tableName: $tableName');
      throw Exception('Error al obtener $plural almacenadas en el dispositivo.\n${e.toString()}');
    }
  }

  Future<void> insertMany(List<T> items) async {
    try {
      final batch = db.batch();

      for (var item in items) {
        batch.insert(tableName, toMap(item));
      }

      await batch.commit(noResult: true);
      
    } catch (e) {
      Logger.error(e.toString(), className: 'LocalData', methodName: 'insertMany', meta: 'tableName: $tableName');
      throw Exception('Error al guardar $plural en el dispositivo.\n${e.toString()}');
    }
  }

  Future<void> insertOne(T item) async {
    try {
      await db.insert(tableName, toMap(item));
    } catch (e) {
      Logger.error(e.toString(), className: 'LocalData', methodName: 'insertOne', meta: 'tableName: $tableName');
      throw Exception('Error al guardar $singular en el dispositivo.}\n${e.toString()}');
    }
  }

  Future<void> update(T item, String? where, List<dynamic>? whereArgs) async {
    try {
      final data = toMap(item);

      if (data.containsKey('updatedAt')) {
        data['updatedAt'] = DateTime.now().millisecondsSinceEpoch;
      }

      await db.update(tableName, data, where: where, whereArgs: whereArgs);

    } catch (e) {
      Logger.error(e.toString(), className: 'LocalData', methodName: 'update', meta: 'tableName: $tableName');
      throw Exception('Error al actualizar $singular en el dispositivo.\n${e.toString()}');
    }
  }
}
