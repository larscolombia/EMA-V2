enum QuestionType {
  open,
  singleChoice,
  trueFalse,
  unknown;

  get name {
    switch (this) {
      case QuestionType.open:
        return 'open_ended';

      case QuestionType.singleChoice:
        return 'single_choice';

      case QuestionType.trueFalse:
        return 'true_false';

      case QuestionType.unknown:
        return 'unknown';

    }
  }

  factory QuestionType.fromMap(Map<String, dynamic> map) {
    return QuestionType.fromName(map['name'] as String);
  }

  factory QuestionType.fromName(String name) {
    final normalized = name.replaceAll('-', '_');

    switch (normalized) {
      case 'open_ended':
        return QuestionType.open;

      case 'single_choice':
      case 'multiple_choice':
        return QuestionType.singleChoice;

      case 'true_false':
        return QuestionType.trueFalse;

      default:
        return QuestionType.unknown;
    }
  }

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'name': name,
    };
  }

  String resumeToFeedback() {
    switch (this) {
      case QuestionType.open:
        return 'de tipo abierta';

      case QuestionType.singleChoice:
        return 'del tipo "seleccionar una única opción"';

      case QuestionType.trueFalse:
        return 'del tipo "seleccionar verdadero o falso';

      default:
        return '';
    }
  }
}
