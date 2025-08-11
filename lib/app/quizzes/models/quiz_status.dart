// ignore_for_file: public_member_api_docs, sort_constructors_first

enum QuizStatus {
  // Con preguntas pendiente de responder
  inProgress,
  
  // Todas las preguntas respondidas
  completed,
  
  // Calificado cuantitativamente
  scored,

  // Calificado cualitativo o con un feedback
  reviewed;

  String get description {
    switch (this) {
      case QuizStatus.inProgress:
        return 'En curso';
      case QuizStatus.completed:
        return 'Completado';
      case QuizStatus.scored:
        return 'Calificado';
      case QuizStatus.reviewed:
        return 'Revisado';
    }
  }
}
