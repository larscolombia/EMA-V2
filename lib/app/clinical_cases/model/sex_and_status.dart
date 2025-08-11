enum SexAndStatus {
  man,
  woman,
  womanPregnantEmbrionaryPhase,
  womanPregnantEarlyFetalPhase,
  womanPregnantMediumFetalPhase,
  womanPregnantLateFetalPhase;

  bool get isPregnant {
    if (this == SexAndStatus.man) return false; 
    if (this == SexAndStatus.woman) return false; 
    return true;
  }
  
  String get description {
    switch (this) {
      case SexAndStatus.man:
        return 'Hombre';
      case SexAndStatus.woman:
        return 'Mujer';
      case SexAndStatus.womanPregnantEmbrionaryPhase:
        return 'Mujer gestante\nFase embrionaria';
      case SexAndStatus.womanPregnantEarlyFetalPhase:
        return 'Mujer gestante\nFetal temprana';
      case SexAndStatus.womanPregnantMediumFetalPhase:
        return 'Mujer gestante\nFetal media';
      case SexAndStatus.womanPregnantLateFetalPhase:
        return 'Mujer gestante\nFetal tardía';
    }
  }

  String get sex {
    if (this == SexAndStatus.man) return 'male';
    return 'female';
  }
 
  double get value {
    switch (this) {
      case SexAndStatus.man:
        return 1.0;
      case SexAndStatus.woman:
        return 2.0;
      case SexAndStatus.womanPregnantEmbrionaryPhase:
        return 3.0;
      case SexAndStatus.womanPregnantEarlyFetalPhase:
        return 4.0;
      case SexAndStatus.womanPregnantMediumFetalPhase:
        return 5.0;
      case SexAndStatus.womanPregnantLateFetalPhase:
        return 6.0;
    }
  }

  factory SexAndStatus.fromValue(double value) {
    switch (value) {
      case 1.0:
        return SexAndStatus.man;
      case 2.0:
        return SexAndStatus.woman;
      case 3.0:
        return SexAndStatus.womanPregnantEmbrionaryPhase;
      case 4.0:
        return SexAndStatus.womanPregnantEarlyFetalPhase;
      case 5.0:
        return SexAndStatus.womanPregnantMediumFetalPhase;
      case 6.0:
        return SexAndStatus.womanPregnantLateFetalPhase;
      default:
        throw Exception('No se reconoce el valor para SexAndStatus');
    }
  }
}
