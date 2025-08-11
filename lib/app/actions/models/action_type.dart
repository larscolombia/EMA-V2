// ignore_for_file: public_member_api_docs, sort_constructors_first

enum ActionType {
  chat,
  clinicalCase,
  // clinicalCaseInteractive,
  quizzes,
  pdf,
  unknown;

  factory ActionType.fromString(String value) {
    switch (value) {
      case 'chat':
        return ActionType.chat;
      case 'clinicalCases':
        return ActionType.clinicalCase;
      // case 'clinicalCaseInteractive':
      //   return ActionType.clinicalCaseInteractive;
      case 'quizzes':
        return ActionType.quizzes;
      case 'pdf':
        return ActionType.pdf;
      default:
        return ActionType.unknown;
    }
  }

  @override
  String toString() {
    switch (this) {
      case ActionType.chat:
        return 'chat';
      case ActionType.clinicalCase:
        return 'clinicalCases';
      // case ActionType.clinicalCaseInteractive:
      //   return 'clinicalCaseInteractive';
      case ActionType.quizzes:
        return 'quizzes';
      case ActionType.pdf:
        return 'pdf';
      default:
        return 'unknown';
    }
  }

  String get title {
    switch (this) {
      case ActionType.chat:
        return 'Chats';
      case ActionType.clinicalCase:
        return 'Casos Clínicos';
      // case ActionType.clinicalCaseInteractive:
      //   return 'Casos Clínicos Interactivos';
      case ActionType.quizzes:
        return 'Cuestionarios';
      case ActionType.pdf:
        return 'PDF';
      default:
        return 'Desconocido';
    }
  }

  String get icon {
    switch (this) {
      case ActionType.chat:
        return 'icon_type_chat';
      case ActionType.clinicalCase:
        return 'icon_type_clinical_cases';
      // case ActionType.clinicalCaseInteractive:
      //   return 'icon_type_clinical_cases';
      case ActionType.quizzes:
        return 'icon_type_quizzes';
      case ActionType.pdf:
        return 'icon_type_pdf';
      default:
        return 'Desconocido';
    }
  }
}
