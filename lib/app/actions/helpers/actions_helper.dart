// ignore_for_file: public_member_api_docs, sort_constructors_first

class ActionsHelper {
  static List<Object> getWhereArgs({
    required int userId,
    int? categoryId,
    String? title,
    String? type,
    int page = 1,
  }) {
    List<Object> whereArgs = [];

    // El id de usuario es obligatorio
    whereArgs.add(userId);

    if (categoryId != null && categoryId != 0) {
      whereArgs.add(categoryId);
    }

    if (type != null && type.isNotEmpty) {
      whereArgs.add(type);
    }

    if (title != null && title.isNotEmpty) {
      whereArgs.add('%$title%');
    }

    return whereArgs;
  }

  static String getWhere({
    required int userId,
    int? categoryId,
    String? title,
    String? type,
    int page = 1,
  }) {
    String where = 'userId = ?';

    if (categoryId != null && categoryId != 0) {
      where += 'categoryId = ?';
    }

    if (type != null && type.isNotEmpty) {
      if (where.isNotEmpty) where += ' AND ';
      where += 'type = ?';
    }

    if (title != null && title.isNotEmpty) {
      if (where.isNotEmpty) where += ' AND ';
      where += 'title LIKE ?';
    }

    return where;
  }
}
