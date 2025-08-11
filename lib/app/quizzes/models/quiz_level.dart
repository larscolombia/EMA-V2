enum QuizzLevel {
  basic,
  normal,
  hard;

  String get name {
    switch (this) {
      case QuizzLevel.basic:
        return 'Básico';
      case QuizzLevel.normal:
        return 'Normal';
      case QuizzLevel.hard:
        return 'Avanzado';
    }
  }

  factory QuizzLevel.fromMap(Map<String, dynamic> map) {
    return QuizzLevel.fromValue(map['value'] as double);
  }

  static QuizzLevel fromValue(double value) {
    switch (value) {
      case 1.0:
        return QuizzLevel.basic;
      case 2.0:
        return QuizzLevel.normal;
      case 3.0:
        return QuizzLevel.hard;
      default:
        return QuizzLevel.normal;
    }
  }

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'name': name,
      'value': value,
    };
  }

  double get value {
    switch (this) {
      case QuizzLevel.basic:
        return 1.0;
      case QuizzLevel.normal:
        return 2.0;
      case QuizzLevel.hard:
        return 3.0;
    }
  }
}
