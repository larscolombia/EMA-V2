 // ignore_for_file: public_member_api_docs, sort_constructors_first
import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/app/quizzes/models/quiz_status.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:uuid/uuid.dart';


class QuizModel {
  final String uid;
  /// Remote test id
  final int? remoteId;
  final int userId;
  final String shortTitle;
  final String title;
  final int numAnswers;
  final int numQuestions;
  final int? score;
  final String feedback;
  final QuizzLevel level;
  final bool animated;
  final DateTime createdAt;
  final DateTime updatedAt;
  final int? categoryId;
  final List<QuestionResponseModel> questions;
  
  QuizModel({
    required this.uid,
    required this.remoteId,
    required this.userId,
    required this.shortTitle,
    required this.title,
    required this.numQuestions,
    required this.createdAt,
    required this.updatedAt,
    this.numAnswers = 0,
    this.score,
    this.feedback = '',
    this.level = QuizzLevel.normal,
    this.animated = false,
    this.categoryId,
    this.questions = const [],
  });

  bool get isEvaluated => feedback.isNotEmpty;

  // bool get animateFeedBack {
  //   if (feedback.isEmpty) return false;
  //   final plazo = updatedAt.add(const Duration(minutes: 1));
  //   return plazo.isBefore(DateTime.now());
  // }

  QuizStatus get status {
    if (feedback.isNotEmpty) return QuizStatus.reviewed;

    if (score != null) return QuizStatus.scored;

    if (numAnswers == numQuestions) return QuizStatus.completed;

    return QuizStatus.inProgress;
  }

  QuizModel copyWith({
    String? uid,
    int? remoteId,
    int? userId,
    QuizStatus? status,
    bool? general,
    String? shortTitle,
    String? title,
    int? numAnswers,
    int? numQuestions,
    int? score,
    String? feedback,
    QuizzLevel? level,
    int? categoryId,
    bool? animated,
    DateTime? createdAt,
    DateTime? updatedAt,
    List<QuestionResponseModel>? questions,
  }) {
    return QuizModel(
      uid: uid ?? this.uid,
      remoteId: remoteId ?? this.remoteId,
      userId: userId ?? this.userId,
      shortTitle: shortTitle ?? this.shortTitle,
      title: title ?? this.title,
      numAnswers: numAnswers ?? this.numAnswers,
      numQuestions: numQuestions ?? this.numQuestions,
      score: score ?? this.score,
      feedback: feedback ?? this.feedback,
      level: level ?? this.level,
      animated: animated ?? this.animated,
      categoryId: categoryId ?? this.categoryId,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      questions: questions ?? this.questions,
    );
  }

  factory QuizModel.generate({
    required int numQuestions,
    required QuizzLevel level,
    required int userId,
    String? categoryName,
    int? categoryId
  }) {
    final title = QuizHelper.title(categoryName, numQuestions, level);
    final shortTitle = QuizHelper.shortTitle(categoryName, numQuestions, level);

    return QuizModel(
      uid: const Uuid().v4(),
      remoteId: 0,
      userId: userId,
      shortTitle: shortTitle,
      title: title,
      numQuestions: numQuestions,
      level: level,
      categoryId: categoryId,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
    );
  }

  factory QuizModel.fromJson(String source) {
    return QuizModel.fromMap(json.decode(source) as Map<String, dynamic>);
  }
  
  // Todo: ajustar con el endpoint
  factory QuizModel.fromApi(Map<String, dynamic> map) {
    return QuizModel(
      uid: map['uid'] as String,
      remoteId: map['remoteId'] != null ? map['remoteId'] as int : null,
      userId: map['userId'] as int,
      shortTitle: map['shortTitle'] as String,
      title: map['title'] as String,
      numQuestions: map['numQuestions'] as int,
      score: map['score'] != null ? map['score'] as int : null,
      feedback: map['feedback'] as String,
      level: QuizzLevel.fromValue(map['level'] as double),
      animated: true,
      createdAt: DateTime.fromMillisecondsSinceEpoch(map['createdAt'] as int),
      updatedAt: DateTime.fromMillisecondsSinceEpoch(map['updatedAt'] as int),
      categoryId: map['categoryId'] != null ? map['categoryId'] as int : null,
    );
  }

  factory QuizModel.fromMap(Map<String, dynamic> map) {
    return QuizModel(
      uid: map['uid'] as String,
      remoteId: map['remoteId'] != null ? map['remoteId'] as int : null,
      userId: map['userId'] as int,
      shortTitle: map['shortTitle'] as String,
      title: map['title'] as String,
      numAnswers: map['numAnswers'] as int,
      numQuestions: map['numQuestions'] as int,
      score: map['score'],
      feedback: map['feedback'] as String,
      level: QuizzLevel.fromValue((map['level'] as int).toDouble()),
      animated: map['animated'] != null ? map['animated'] as int == 1 : false,
      categoryId: map['categoryId'] != null ? map['categoryId'] as int : null,
      createdAt: DateTime.fromMillisecondsSinceEpoch(map['createdAt'] as int),
      updatedAt: DateTime.fromMillisecondsSinceEpoch(map['updatedAt'] as int),
    );
  }

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'uid': uid,
      'remoteId': remoteId ?? 0,
      'userId': userId,
      'shortTitle': shortTitle,
      'title': title,
      'numQuestions': numQuestions,
      'numAnswers': numAnswers,
      'score' : score,
      'feedback': feedback,
      'level': level.value.toInt(),
      'animated': animated ? 1 : 0,
      'createdAt': createdAt.millisecondsSinceEpoch,
      'updatedAt': updatedAt.millisecondsSinceEpoch,
      'categoryId': categoryId ?? 0,
    };
  }

  Map<String, dynamic> toEvaluationMap() {
    return <String, dynamic>{
      'uid': uid,
      'thread': '',
      'userId': userId,
      'test_id': remoteId,
      'questions': questions.map((e) => e.toEvaluationMap()).toList(),
    };
  }

  Map<String, dynamic> toRequestBody() {
    final body = <String, dynamic>{
      'id_categoria': categoryId != null ? [categoryId] : [],
      'num_questions': numQuestions,
      'nivel': level.name
    };
    print('DEBUG: QuizModel.toRequestBody - body: $body');
    return body;
  }

  String toJson() => json.encode(toMap());

  String resumeToFeedback() {
    final questionsResume = questions.map((q) => q.resumeToFeedback()).join('\n');
    return '$title\n$questionsResume';
  }
}
