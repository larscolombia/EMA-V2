

class AnswerModel {
  final String answer;
  final List<String> answers;

  static final List<String> letters = ['A', 'B', 'C', 'D', 'E'];

  AnswerModel._({
    required this.answer,
    required this.answers,
  });

  // Factory constructor for open-ended questions
  factory AnswerModel.openEnded(String answer) {
    final cleanAnswer = answer.trim();
    if (cleanAnswer.isEmpty) throw ArgumentError('Debes responder la pregunta.');

    return AnswerModel._(
      answer: cleanAnswer,
      answers: [],
    );
  }

  // Factory constructor for multiple-choice questions
  factory AnswerModel.multipleChoice(List<String> answers, List<String> options) {
    if (answers.isEmpty) throw ArgumentError('Debes seleccionar al menos una opción.');
    if (options.isEmpty) throw ArgumentError('No se encontrarion opciones para la pregunta.');
    if (answers.length > options.length) throw ArgumentError('Se encontraron más respuestas que opciones.');

    for (var answer in answers) {
      if (!letters.contains(answer)) {
        throw ArgumentError('La $answer no está dentro de las opcoines: ${letters.join(", ")}');
      }
    }
    
    // Generamos un nuevo array combinando las letras con sus valores correspondientes de options
    List<String> combinedAnswers = answers.map((letter) {
      int index = letters.indexOf(letter);

      if (index == -1 || index >= options.length) {
        throw ArgumentError('La respuesta debe tener alguna de estas letras: ${letters.join(", ")}');
      }

      return options[index];
    }).toList();

    // Retornamos una nueva instancia con las respuestas combinadas
    return AnswerModel._(
      answer: '',
      answers: combinedAnswers,
    );
  }

  // Factory constructor for single-choice questions
  factory AnswerModel.singleChoice(String answer, List<String> options) {
    final cleanAnswer = answer.trim();
    if (cleanAnswer.isEmpty) throw ArgumentError('Debes seleccionar una opción.');

    // Verificamos que la letra proporcionada sea válida
    if (!letters.contains(answer)) {
      throw ArgumentError('La respuesta debe tener alguna de estas letras: ${letters.join(", ")}');
    }

    // Obtenemos el índice de la letra
    int index = letters.indexOf(answer);

    // Generamos el texto para answer asegurándonos de que el índice no excede las opciones
    String generatedText = index < options.length
        ? options[index]
        : 'Invalid index';

    // Retornamos una nueva instancia configurada correctamente
    return AnswerModel._(
      answer: generatedText,
      answers: [],
    );
  }

  // Factory constructor for true/false questions
  factory AnswerModel.trueFalse(bool answer) {
    return AnswerModel._(
      answer: answer.toString(),
      answers: [],
    );
  }
}
