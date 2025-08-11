// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/actions/models/action_type.dart';
import 'package:uuid/uuid.dart';


class ActionModel {
  final String id;
  final int userId;
  final String itemId;
  final String shortTitle;
  final String title;
  final ActionType type;
  final DateTime createdAt;
  final DateTime updatedAt;
  final int? categoryId;

  ActionModel._({
    required this.id,
    required this.userId,
    required this.itemId,
    required this.shortTitle,
    required this.title,
    required this.type,
    required this.createdAt,
    required this.updatedAt,
    this.categoryId,
  });

  factory ActionModel.chat(int userId, String itemId, String title) {
    return ActionModel._(
      id: const Uuid().v4(),
      userId: userId,
      itemId: itemId,
      shortTitle: title,
      title: title,
      type: ActionType.chat,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      categoryId: 0,
    );
  }

  factory ActionModel.clinicalCase(int userId, String itemId, String shortTitle, String title) {
    return ActionModel._(
      id: const Uuid().v4(),
      userId: userId,
      itemId: itemId,
      shortTitle: shortTitle,
      title: title,
      type: ActionType.clinicalCase,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      categoryId: null,
    );
  }

  factory ActionModel.quizzes(int userId, String itemId, String shortTitle, String title, int? categoryId) {
    return ActionModel._(
      id: const Uuid().v4(),
      userId: userId,
      itemId: itemId,
      shortTitle: shortTitle,
      title: title,
      type: ActionType.quizzes,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      categoryId: categoryId ?? 0,
    );
  }

  factory ActionModel.fromMap(Map<String, dynamic> map) {
    return ActionModel._(
      id: map['id'] as String,
      userId: map['userId'] as int,
      itemId: map['itemId'] as String,
      shortTitle: map['shortTitle'] as String,
      title: map['title'] as String,
      type: ActionType.fromString(map['type'] as String),
      createdAt: DateTime.fromMillisecondsSinceEpoch(map['createdAt'] as int),
      updatedAt: DateTime.fromMillisecondsSinceEpoch(map['updatedAt'] as int),
      categoryId: map['categoryId'] != null ? map['categoryId'] as int : null,
    );
  }

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'id': id,
      'userId': userId,
      'itemId': itemId,
      'shortTitle': shortTitle,
      'title': title,
      'type': type.toString(),
      'createdAt': createdAt.millisecondsSinceEpoch,
      'updatedAt': updatedAt.millisecondsSinceEpoch,
      'categoryId': categoryId,
    };
  }
}
