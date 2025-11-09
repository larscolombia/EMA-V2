// ignore_for_file: public_member_api_docs, sort_constructors_first
import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/app/quizzes/helper/question_helper.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_type.dart';
import 'package:uuid/uuid.dart';

class QuestionResponseModel {
  final String id;

  /// El identificador de la pregunta en el servidor
  final int remoteId;
  final String
  quizId; // Se cambiará por parentId para vincularla con chat y con casos clínicos
  final String parentType;
  final String message;
  final String question;

  /// Respuesta correcta
  final String? answer;
  final String? userAnswer;
  final List<String> userAnswers;
  final QuestionType type;
  final List<String> options;
  final bool? isCorrect;
  final String? fit;
  final DateTime createdAt;
  final DateTime updatedAt;

  QuestionResponseModel({
    required this.id,
    required this.remoteId,
    required this.quizId,
    required this.question,
    required this.answer,
    this.userAnswer,
    this.userAnswers = const [],
    required this.type,
    this.options = const [],
    this.isCorrect,
    this.fit,
    required this.createdAt,
    required this.updatedAt,
    // New properties for clinical cases
    this.parentType = 'quiz',
    this.message = '',
  });

  get isAnswered {
    // En caso de añadir un tipo de pregunta
    // de selección múltiple, se debe verificar
    // si la respuesta es un array vacío
    return userAnswer != null && userAnswer!.isNotEmpty;
  }

  // Se usa para mostrar la respuesta del usuario en el chat
  String get answerdString {
    if (type == QuestionType.open) {
      return userAnswer.toString();
    }

    if (type == QuestionType.singleChoice) {
      return userAnswer.toString();
    }

    if (type == QuestionType.trueFalse) {
      return userAnswer == 'true' ? 'Verdadero' : 'Falso';
    }

    return '';
  }

  QuestionResponseModel copyWith({
    String? id,
    String? userAnswer,
    List<String>? userAnswers,
    List<String>? options,
    bool? isCorrect,
    bool updateIsCorrect = false, // Flag para forzar actualización de isCorrect
    String? fit,
    DateTime? createdAt,
    DateTime? updatedAt,
    String? parentType,
    String? message,
  }) {
    return QuestionResponseModel(
      id: id ?? this.id,
      remoteId: remoteId,
      quizId: quizId,
      question: question,
      answer: answer,
      userAnswer: userAnswer ?? this.userAnswer,
      userAnswers: userAnswers ?? this.userAnswers,
      type: type,
      options: options ?? this.options,
      isCorrect: updateIsCorrect ? isCorrect : (isCorrect ?? this.isCorrect),
      fit: fit ?? this.fit,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      parentType: parentType ?? this.parentType,
      message: message ?? this.message,
    );
  }

  factory QuestionResponseModel.empty({
    String? parentType,
    String? message,
    String? quizId,
  }) {
    return QuestionResponseModel(
      id: Uuid().v4(),
      remoteId: 0,
      quizId: quizId ?? '',
      question: '',
      answer: '',
      userAnswer: '',
      userAnswers: [],
      type: QuestionType.open,
      options: [],
      isCorrect: false,
      fit: '',
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      parentType: parentType ?? 'quiz',
      message: message ?? '',
    );
  }

  factory QuestionResponseModel.fromApi(Map<String, dynamic> map) {
    final options = QuestionHelper.getOptions(map);

    return QuestionResponseModel(
      id: Uuid().v4(),
      remoteId: map['id'] as int,
      quizId: map['quizId'] as String,
      question: map['question'] as String,
      answer: map['answer'] != null ? map['answer'] as String : '',
      type: QuestionType.fromName(map['type'] as String),
      options: options,
      isCorrect: map['isCorrect'] != null ? map['isCorrect'] == 1 : null,
      fit: map['fit'] == null ? '' : map['fit'] as String,
      createdAt:
          map['created_at'] != null
              ? DateTime.parse(map['created_at'] as String)
              : DateTime.now(),
      updatedAt:
          map['updated_at'] != null
              ? DateTime.parse(map['updated_at'] as String)
              : DateTime.now(),
    );
  }

  factory QuestionResponseModel.fromClinicalCaseApi({
    required String quizId,
    required String feedback,
    required Map<String, dynamic> questionMap,
  }) {
    final options = QuestionHelper.getOptions(questionMap);

    final rawType =
        questionMap['type'] ??
        questionMap['question_type'] ??
        questionMap['tipo'] ??
        '';

    final typeName =
        rawType is String ? rawType.replaceAll('-', '_') : rawType.toString();

    return QuestionResponseModel(
      id: Uuid().v4(),
      remoteId:
          questionMap['id'] is int
              ? questionMap['id'] as int
              : int.tryParse(questionMap['id']?.toString() ?? '') ?? 0,
      quizId: quizId,
      question:
          (questionMap['question'] ?? questionMap['texto'] ?? '') as String,
      answer:
          questionMap['answer'] != null ? questionMap['answer'] as String : '',
      type: QuestionType.fromName(typeName),
      options: options,
      isCorrect:
          questionMap['isCorrect'] != null
              ? questionMap['isCorrect'] == 1
              : null,
      fit: feedback,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      // Nuevos campos para casos clínicos
      parentType: 'clinical_case',
    );
  }

  factory QuestionResponseModel.fromMap(Map<String, dynamic> map) {
    return QuestionResponseModel(
      id: map['id'] as String,
      remoteId: map['remoteId'] as int,
      quizId: map['quizId'] as String,
      parentType: map['parentType'] as String,
      question: map['question'] as String,
      answer: map['answer'] as String,
      userAnswer: map['userAnswer'] as String,
      userAnswers: List<String>.from(jsonDecode(map['userAnswers'] as String)),
      type: QuestionType.fromName(map['type'] as String),
      options: List<String>.from(jsonDecode(map['options'] as String)),
      isCorrect: map['isCorrect'] != null ? map['isCorrect'] == 1 : null,
      fit: map['fit'] == null ? '' : map['fit'] as String,
      createdAt: DateTime.fromMillisecondsSinceEpoch(map['createdAt'] as int),
      updatedAt: DateTime.fromMillisecondsSinceEpoch(map['updatedAt'] as int),
    );
  }

  Map<String, dynamic> toEvaluationMap() {
    return <String, dynamic>{
      'question_id': remoteId,
      'answer': userAnswer ?? '',
      'type': type.name,
    };
  }

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'id': id,
      'remoteId': remoteId,
      'quizId': quizId,
      'parentType': parentType,
      'question': question,
      'answer': answer,
      'userAnswer': userAnswer ?? '',
      'userAnswers': jsonEncode(userAnswers),
      'type': type.name,
      'options': jsonEncode(options),
      'isCorrect': isCorrect == null ? null : (isCorrect! ? 1 : 0),
      'fit': fit ?? '',
      'createdAt': createdAt.millisecondsSinceEpoch,
      'updatedAt': updatedAt.millisecondsSinceEpoch,
    };
  }

  String toAiQuestionMessage() {
    String resume = '$question  \n';

    const letters = ['A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J'];
    var counter = 0;

    for (var option in options) {
      resume += ' * ${letters[counter]} - $option  \n';
      counter++;
    }

    return resume;
  }

  String toUserAnswerMessage() {
    // const letters = ['A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J'];
    // final index = options.indexOf(userAnswer ?? '');
    return answerdString;
  }

  String resumeToFeedback() {
    final optionsStr =
        options.isNotEmpty ? ' con las opciones ${options.join(', ')}' : '';

    String resume =
        'A la pregunta: "$question" ${type.resumeToFeedback()} $optionsStr ';
    resume += 'el usuario respondió: "$answerdString" ';
    resume += 'Recibió como respuesta del sistema: "$fit"';
    return resume;
  }

  @override
  String toString() {
    return 'id: $id - remoteId: $remoteId\n question: $question\nanswer: $answer - userAnswer: $userAnswer\nfit: $fit, $updatedAt)';
  }
}
