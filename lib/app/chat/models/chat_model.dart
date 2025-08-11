// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:uuid/uuid.dart';

class ChatModel {
  String uid;
  String threadId;
  int userId;
  String shortTitle;
  DateTime createdAt;
  DateTime updatedAt;

  ChatModel({
    required this.uid,
    required this.threadId,
    required this.userId,
    required this.shortTitle,
    required this.createdAt,
    required this.updatedAt,
  });

  ChatModel copyWith({
    int? userId,
    String? threadId,
    String? shortTitle,
    String? uid,
  }) {
    return ChatModel(
      uid: uid ?? this.uid,
      userId: userId ?? this.userId,
      threadId: threadId ?? this.threadId,
      shortTitle: shortTitle ?? this.shortTitle,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }

  factory ChatModel.fromApi(Map<String, dynamic> map) {
    return ChatModel(
      uid: map['uid'] as String,
      threadId: map['threadId'] as String,
      userId: map['userId'] as int,
      shortTitle: map['shortTitle'] as String,
      createdAt: DateTime.fromMillisecondsSinceEpoch(map['createdAt'] as int),
      updatedAt: DateTime.fromMillisecondsSinceEpoch(map['updatedAt'] as int),
    );
  }

  factory ChatModel.empty([int? userId]) {
    return ChatModel(
      uid: Uuid().v4(),
      threadId: '',
      userId: userId ?? 0,
      shortTitle: '',
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
    );
  }

  factory ChatModel.fromMap(Map<String, dynamic> map) {
    return ChatModel(
      uid: map['uid'] as String,
      threadId: map['threadId'] as String,
      userId: map['userId'] as int,
      shortTitle: map['shortTitle'] as String,
      createdAt: DateTime.fromMillisecondsSinceEpoch(map['createdAt'] as int),
      updatedAt: DateTime.fromMillisecondsSinceEpoch(map['updatedAt'] as int),
    );
  }

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'uid': uid,
      'threadId': threadId,
      'userId': userId,
      'shortTitle': shortTitle,
      'createdAt': createdAt.millisecondsSinceEpoch,
      'updatedAt': updatedAt.millisecondsSinceEpoch,
    };
  }
}
