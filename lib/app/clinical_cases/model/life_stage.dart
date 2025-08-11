enum LifeStage {
  prenatal,
  neonato,
  lactante,
  infante,
  preescolar,
  escolar,
  adolescente,
  joven,
  adulto,
  adultoMayor,
  anciano;

  /// Identico a la definisi´no en Laravel 
  /// Models ClinicalCase AGE_GROUPS
  String get age {
    switch (this) {
      case LifeStage.prenatal:
        return 'prenatal';
      case LifeStage.neonato:
        return 'neonato';
      case LifeStage.lactante:
        return 'lactante';
      case LifeStage.infante:
        return 'infante';
      case LifeStage.preescolar:
        return 'preescolar';
      case LifeStage.escolar:
        return 'escolar';
      case LifeStage.adolescente:
        return 'adolescente';
      case LifeStage.joven:
        return 'joven';
      case LifeStage.adulto:
        return 'adulto';
      case LifeStage.adultoMayor:
        return 'adulto_mayor';
      case LifeStage.anciano:
        return 'anciano';
    }
  }

  String get name {
    switch (this) {
      case LifeStage.prenatal:
        return 'Prenatal';
      case LifeStage.neonato:
        return 'Neonato';
      case LifeStage.lactante:
        return 'Lactante';
      case LifeStage.infante:
        return 'Infante';
      case LifeStage.preescolar:
        return 'Preescolar';
      case LifeStage.escolar:
        return 'Escolar';
      case LifeStage.adolescente:
        return 'Adolescente';
      case LifeStage.joven:
        return 'Joven';
      case LifeStage.adulto:
        return 'Adulto';
      case LifeStage.adultoMayor:
        return 'Adulto mayor';
      case LifeStage.anciano:
        return 'Anciano';
    }
  }

  String get description {
    switch (this) {
      case LifeStage.prenatal:
        return 'Antes de nacer';
      case LifeStage.neonato:
        return '0 - 28 días';
      case LifeStage.lactante:
        return '29 días - 1 año';
      case LifeStage.infante:
        return '1 - 3 años';
      case LifeStage.preescolar:
        return '3 - 5 años';
      case LifeStage.escolar:
        return '6 - 12 años';
      case LifeStage.adolescente:
        return '13 - 18 años';
      case LifeStage.joven:
        return '19 - 30 años';
      case LifeStage.adulto:
        return '31 - 60 años';
      case LifeStage.adultoMayor:
        return '61 - 74 años';
      case LifeStage.anciano:
        return '75 o más años';
    }
  }

  double get value {
    switch (this) {
      case LifeStage.prenatal:
        return 1.0;
      case LifeStage.neonato:
        return 2.0;
      case LifeStage.lactante:
        return 3.0;
      case LifeStage.infante:
        return 4.0;
      case LifeStage.preescolar:
        return 5.0;
      case LifeStage.escolar:
        return 6.0;
      case LifeStage.adolescente:
        return 7.0;
      case LifeStage.joven:
        return 8.0;
      case LifeStage.adulto:
        return 9.0;
      case LifeStage.adultoMayor:
        return 10.0;
      case LifeStage.anciano:
        return 11.0;
    }
  }

  bool get maybePregnant {
    if (this == LifeStage.prenatal) return false;
    if (this == LifeStage.neonato) return false;
    if (this == LifeStage.lactante) return false;
    if (this == LifeStage.infante) return false;
    if (this == LifeStage.preescolar) return false;
    
    return true;
  }

  factory LifeStage.fromValue(double value) {
    switch (value) {
      case 1.0:
        return LifeStage.prenatal;
      case 2.0:
        return LifeStage.neonato;
      case 3.0:
        return LifeStage.lactante;
      case 4.0:
        return LifeStage.infante;
      case 5.0:
        return LifeStage.preescolar;
      case 6.0:
        return LifeStage.escolar;
      case 7.0:
        return LifeStage.adolescente;
      case 8.0:
        return LifeStage.joven;
      case 9.0:
        return LifeStage.adulto;
      case 10.0:
        return LifeStage.adultoMayor;
      case 11.0:
        return LifeStage.anciano;
      default:
        return LifeStage.adulto;
    }
  }
}
